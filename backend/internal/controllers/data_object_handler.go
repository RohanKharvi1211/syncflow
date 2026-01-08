package controllers

import (
	"encoding/json"
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DataObjectHandler struct {
	access *ControllerAccess
}

func NewDataObjectHandler() *DataObjectHandler {
	return &DataObjectHandler{}
}

func NewDataObjectHandlerWithAccess(access *ControllerAccess) *DataObjectHandler {
	return &DataObjectHandler{access: access}
}

// GetDataObjects returns all data objects for connections in user's company
func (h *DataObjectHandler) GetDataObjects(c *gin.Context) {
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

	// Optional: filter by connection_id if provided
	connectionIDStr := c.Query("connection_id")
	var dataObjects []models.DataObject
	query := config.DB.
		Joins("JOIN connections ON data_objects.connection_id = connections.id").
		Where("connections.company_id = ?", companyIDUUID).
		Preload("Connection.App")

	if connectionIDStr != "" {
		connectionID, err := uuid.Parse(connectionIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid connection_id",
			})
			return
		}
		query = query.Where("data_objects.connection_id = ?", connectionID)
	}

	if err := query.Find(&dataObjects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch data objects",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data_objects": dataObjects,
	})
}

// GetDataObject returns a specific data object
func (h *DataObjectHandler) GetDataObject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid data object ID",
		})
		return
	}

	var dataObject models.DataObject
	if err := config.DB.
		Preload("Connection.App").
		First(&dataObject, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Data object not found",
		})
		return
	}

	c.JSON(http.StatusOK, dataObject)
}

// CreateDataObjectRequest represents the request body for creating a data object
type CreateDataObjectRequest struct {
	ConnectionID uuid.UUID              `json:"connection_id" binding:"required"`
	ObjectType   string                 `json:"object_type" binding:"required"` // 'SHEET', 'TABLE', 'INVOICE', 'ENTITY'
	Identifier   string                 `json:"identifier" binding:"required"`   // sheet name, table name, object API name
	Config       map[string]interface{} `json:"config"`                          // range, columns, filters
}

// CreateDataObject creates a new data object
func (h *DataObjectHandler) CreateDataObject(c *gin.Context) {
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

	var req CreateDataObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid data object data",
			"details": err.Error(),
		})
		return
	}

	// Validate connection exists and belongs to user's company
	var baseConnection models.Connection
	if err := config.DB.Where("id = ? AND company_id = ?", req.ConnectionID, companyIDUUID).Preload("App").First(&baseConnection).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found or access denied",
		})
		return
	}

	// For Google Sheets/Drive, create a new connection entry for each sheet
	// This allows each sheet to appear as a separate connection in the UI
	var connectionToUse models.Connection
	if baseConnection.App.Name == "googlesheet" || baseConnection.App.Name == "googledrive" {
		// Check if a connection already exists for this specific sheet (identifier)
		var existingSheetConnection models.Connection
		err := config.DB.Where("company_id = ? AND app_id = ? AND provider = ? AND provider_user_id = ?",
			companyIDUUID, baseConnection.AppID, baseConnection.Provider, baseConnection.ProviderUserID).
			Joins("JOIN data_objects ON data_objects.connection_id = connections.id").
			Where("data_objects.identifier = ? AND data_objects.object_type = ?", req.Identifier, req.ObjectType).
			First(&existingSheetConnection).Error

		if err == nil {
			// Connection for this sheet already exists, use it
			connectionToUse = existingSheetConnection
		} else {
			// Create a new connection entry for this sheet (shares OAuth credentials)
			newConnection := models.Connection{
				UserID:         baseConnection.UserID,
				CompanyID:      baseConnection.CompanyID,
				AppID:          baseConnection.AppID,
				Provider:       baseConnection.Provider,
				Type:           baseConnection.Type,
				AuthType:       baseConnection.AuthType,
				AccessToken:    baseConnection.AccessToken,    // Share OAuth credentials
				RefreshToken:   baseConnection.RefreshToken,  // Share OAuth credentials
				TokenExpiresAt: baseConnection.TokenExpiresAt,
				ProviderUserID: baseConnection.ProviderUserID,
				Status:         baseConnection.Status,
				IsActive:       true,
			}
			if err := config.DB.Create(&newConnection).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to create connection for sheet",
					"details": err.Error(),
				})
				return
			}
			connectionToUse = newConnection
		}
	} else {
		// For other apps, use the existing connection
		connectionToUse = baseConnection
	}

	// Marshal config
	configJSON := "{}"
	if req.Config != nil {
		configBytes, err := json.Marshal(req.Config)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid config format",
				"details": err.Error(),
			})
			return
		}
		configJSON = string(configBytes)
	}

	// Create data object linked to the connection (new or existing)
	dataObject := models.DataObject{
		ConnectionID: connectionToUse.ID,
		ObjectType:   req.ObjectType,
		Identifier:   req.Identifier,
		Config:       configJSON,
	}

	if err := config.DB.Create(&dataObject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create data object",
			"details": err.Error(),
		})
		return
	}

	// Load relations for response
	config.DB.Preload("Connection.App").First(&dataObject, dataObject.ID)

	c.JSON(http.StatusCreated, dataObject)
}

// UpdateDataObject updates an existing data object
func (h *DataObjectHandler) UpdateDataObject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid data object ID",
		})
		return
	}

	var dataObject models.DataObject
	if err := config.DB.First(&dataObject, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Data object not found",
		})
		return
	}

	var req struct {
		ObjectType string                 `json:"object_type"`
		Identifier string                 `json:"identifier"`
		Config     map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid data object data",
			"details": err.Error(),
		})
		return
	}

	// Update fields
	if req.ObjectType != "" {
		dataObject.ObjectType = req.ObjectType
	}
	if req.Identifier != "" {
		dataObject.Identifier = req.Identifier
	}
	if req.Config != nil {
		configBytes, _ := json.Marshal(req.Config)
		dataObject.Config = string(configBytes)
	}

	if err := config.DB.Save(&dataObject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update data object",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dataObject)
}

// DeleteDataObject deletes a data object (soft delete)
func (h *DataObjectHandler) DeleteDataObject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid data object ID",
		})
		return
	}

	if err := config.DB.Delete(&models.DataObject{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete data object",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data object deleted successfully",
	})
}

