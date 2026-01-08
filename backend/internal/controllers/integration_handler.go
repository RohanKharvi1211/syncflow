package controllers

import (
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IntegrationHandler struct {
	access *ControllerAccess
}

func NewIntegrationHandler() ControllerIntegrationMethods {
	return &IntegrationHandler{}
}

func NewIntegrationHandlerWithAccess(access *ControllerAccess) ControllerIntegrationMethods {
	return &IntegrationHandler{access: access}
}

// GetIntegrations returns all integrations for a company
func (h *IntegrationHandler) GetIntegrations(c *gin.Context) {
	companyIDStr := c.Query("company_id")
	if companyIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "company_id is required",
		})
		return
	}

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company_id",
		})
		return
	}

	var integrations []models.Integration
	if err := config.DB.Where("company_id = ?", companyID).
		Preload("SourceApp").
		Preload("DestinationApp").
		Preload("SourceConnection").
		Preload("DestinationConnection").
		Find(&integrations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch integrations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"integrations": integrations,
	})
}

// GetIntegration returns a specific integration
func (h *IntegrationHandler) GetIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid integration ID",
		})
		return
	}

	var integration models.Integration
	if err := config.DB.Preload("Company").
		Preload("SourceApp").
		Preload("DestinationApp").
		Preload("SourceConnection").
		Preload("DestinationConnection").
		Preload("SyncJobs").
		First(&integration, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Integration not found",
		})
		return
	}

	c.JSON(http.StatusOK, integration)
}

// CreateIntegration creates a new integration
func (h *IntegrationHandler) CreateIntegration(c *gin.Context) {
	var integration models.Integration
	if err := c.ShouldBindJSON(&integration); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid integration data",
			"details": err.Error(),
		})
		return
	}

	if err := config.DB.Create(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create integration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, integration)
}

// UpdateIntegration updates an existing integration
func (h *IntegrationHandler) UpdateIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid integration ID",
		})
		return
	}

	var integration models.Integration
	if err := config.DB.First(&integration, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Integration not found",
		})
		return
	}

	if err := c.ShouldBindJSON(&integration); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid integration data",
			"details": err.Error(),
		})
		return
	}

	if err := config.DB.Save(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update integration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, integration)
}

// DeleteIntegration deletes an integration
func (h *IntegrationHandler) DeleteIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid integration ID",
		})
		return
	}

	if err := config.DB.Delete(&models.Integration{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete integration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Integration deleted successfully",
	})
}

