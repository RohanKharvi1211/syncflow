package controllers

import (
	"encoding/json"
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AppHandler struct {
	access *ControllerAccess
}

func NewAppHandler() ControllerAppMethods {
	return &AppHandler{}
}

func NewAppHandlerWithAccess(access *ControllerAccess) ControllerAppMethods {
	return &AppHandler{access: access}
}

// GetApps returns all apps
func (h *AppHandler) GetApps(c *gin.Context) {
	var apps []models.App
	query := config.DB.Where("is_active = ?", true)

	// Filter by type if provided
	// Supports: "source" (returns source and both), "destination" (returns destination and both), or exact match
	if appType := c.Query("type"); appType != "" {
		if appType == "source" {
			// Return apps that can be used as source (type = 'source' OR type = 'both')
			query = query.Where("type IN ?", []string{"source", "both"})
		} else if appType == "destination" {
			// Return apps that can be used as destination (type = 'destination' OR type = 'both')
			query = query.Where("type IN ?", []string{"destination", "both"})
		} else {
			// Exact match for 'both' or other specific types
			query = query.Where("type = ?", appType)
		}
	}

	if err := query.Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch apps",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"apps": apps,
	})
}

// GetApp returns a specific app
func (h *AppHandler) GetApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app ID",
		})
		return
	}

	var app models.App
	if err := config.DB.First(&app, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "App not found",
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// CreateAppRequest represents the request for creating an app
type CreateAppRequest struct {
	Name          string                 `json:"name" binding:"required"`
	DisplayName   string                 `json:"display_name" binding:"required"`
	Description   string                 `json:"description"`
	Type          string                 `json:"type" binding:"required"` // 'source', 'destination', or 'both'
	MetadataSchema map[string]interface{} `json:"metadata_schema" binding:"required"`
}

// CreateApp creates a new app
func (h *AppHandler) CreateApp(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app data",
			"details": err.Error(),
		})
		return
	}

	// Validate type
	if req.Type != "source" && req.Type != "destination" && req.Type != "both" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Type must be 'source', 'destination', or 'both'",
		})
		return
	}

	// Convert metadata schema to JSON string
	metadataSchemaJSON, err := json.Marshal(req.MetadataSchema)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid metadata schema format",
			"details": err.Error(),
		})
		return
	}

	app := models.App{
		Name:          req.Name,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		Type:          req.Type,
		MetadataSchema: string(metadataSchemaJSON),
		IsActive:      true,
	}

	if err := config.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create app",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, app)
}

// UpdateApp updates an existing app
func (h *AppHandler) UpdateApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app ID",
		})
		return
	}

	var app models.App
	if err := config.DB.First(&app, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "App not found",
		})
		return
	}

	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app data",
			"details": err.Error(),
		})
		return
	}

	// Update fields
	app.DisplayName = req.DisplayName
	app.Description = req.Description
	app.Type = req.Type

	metadataSchemaJSON, err := json.Marshal(req.MetadataSchema)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid metadata schema format",
			"details": err.Error(),
		})
		return
	}
	app.MetadataSchema = string(metadataSchemaJSON)

	if err := config.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update app",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// DeleteApp deletes an app (soft delete)
func (h *AppHandler) DeleteApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app ID",
		})
		return
	}

	if err := config.DB.Delete(&models.App{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete app",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App deleted successfully",
	})
}

