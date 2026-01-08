package controllers

import (
	"encoding/json"
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ConnectionHandler struct {
	access *ControllerAccess
}

func NewConnectionHandler() ControllerConnectionMethods {
	return &ConnectionHandler{}
}

func NewConnectionHandlerWithAccess(access *ControllerAccess) ControllerConnectionMethods {
	return &ConnectionHandler{access: access}
}

// GetConnections returns all connections for a company (from auth context)
func (h *ConnectionHandler) GetConnections(c *gin.Context) {
	// Get company_id from auth context (set by auth middleware)
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	companyIDUUID, ok := companyID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company_id in context",
		})
		return
	}

	var connections []models.Connection
	if err := config.DB.Where("company_id = ?", companyIDUUID).
		Preload("App").
		Preload("Metadata").
		Preload("DataObjects"). // Include data objects to show sheet names
		Find(&connections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch connections",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connections": connections,
	})
}

// GetConnection returns a specific connection (validates company ownership)
func (h *ConnectionHandler) GetConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection ID",
		})
		return
	}

	// Get company_id from auth context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	companyIDUUID, ok := companyID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company_id in context",
		})
		return
	}

	var connection models.Connection
	if err := config.DB.Preload("Company").Preload("App").Preload("Metadata").First(&connection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found",
		})
		return
	}

	// Verify connection belongs to user's company
	if connection.CompanyID != companyIDUUID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	c.JSON(http.StatusOK, connection)
}

// CreateConnectionRequest represents the request body for creating a connection
type CreateConnectionRequest struct {
	CompanyID    uuid.UUID              `json:"company_id" binding:"required"`
	AppID        uuid.UUID              `json:"app_id" binding:"required"`
	AccessToken  string                 `json:"access_token" binding:"required"`
	RefreshToken string                 `json:"refresh_token" binding:"required"`
	Metadata     map[string]interface{} `json:"metadata" binding:"required"`
}

// CreateConnection creates a new connection with metadata validation
func (h *ConnectionHandler) CreateConnection(c *gin.Context) {
	// Get company_id from auth context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	companyIDUUID, ok := companyID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company_id in context",
		})
		return
	}

	var req CreateConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection data",
			"details": err.Error(),
		})
		return
	}

	// Override company_id from auth context (security)
	req.CompanyID = companyIDUUID

	// Fetch the app to get metadata schema
	var app models.App
	if err := config.DB.First(&app, req.AppID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "App not found",
		})
		return
	}

	// Validate metadata against app's schema
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid metadata format",
			"details": err.Error(),
		})
		return
	}

	if err := utils.ValidateMetadata(string(metadataJSON), app.MetadataSchema); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Metadata validation failed",
			"details": err.Error(),
		})
		return
	}

	// Create connection
	connection := models.Connection{
		CompanyID:    req.CompanyID,
		AppID:        req.AppID,
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
		IsActive:     true,
	}

	// Start transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&connection).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create connection",
			"details": err.Error(),
		})
		return
	}

	// Create metadata
	metadata := models.Metadata{
		ConnectionID: connection.ID,
		Data:         string(metadataJSON),
	}

	if err := tx.Create(&metadata).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create metadata",
			"details": err.Error(),
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save connection",
			"details": err.Error(),
		})
		return
	}

	// Load relations for response
	config.DB.Preload("App").Preload("Metadata").First(&connection, connection.ID)

	c.JSON(http.StatusCreated, connection)
}

// UpdateConnection updates an existing connection (e.g., refresh token)
func (h *ConnectionHandler) UpdateConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection ID",
		})
		return
	}

	// Get company_id from auth context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	companyIDUUID, ok := companyID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company_id in context",
		})
		return
	}

	var connection models.Connection
	if err := config.DB.First(&connection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found",
		})
		return
	}

	// Verify connection belongs to user's company
	if connection.CompanyID != companyIDUUID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if err := c.ShouldBindJSON(&connection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection data",
			"details": err.Error(),
		})
		return
	}

	if err := config.DB.Save(&connection).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update connection",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, connection)
}

// DeleteConnection deletes a connection
func (h *ConnectionHandler) DeleteConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection ID",
		})
		return
	}

	// Get company_id from auth context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	companyIDUUID, ok := companyID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid company_id in context",
		})
		return
	}

	// Verify connection belongs to user's company before deleting
	var connection models.Connection
	if err := config.DB.First(&connection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found",
		})
		return
	}

	if connection.CompanyID != companyIDUUID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if err := config.DB.Delete(&models.Connection{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete connection",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Connection deleted successfully",
	})
}

