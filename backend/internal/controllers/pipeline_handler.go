package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PipelineHandler struct {
	access           *ControllerAccess
	pipelineService  *services.PipelineService
}

func NewPipelineHandler() *PipelineHandler {
	return &PipelineHandler{
		pipelineService: services.NewPipelineService(),
	}
}

func NewPipelineHandlerWithAccess(access *ControllerAccess) *PipelineHandler {
	return &PipelineHandler{
		access:          access,
		pipelineService: services.NewPipelineService(),
	}
}

// GetPipelines returns all pipelines for a company (from auth context)
func (h *PipelineHandler) GetPipelines(c *gin.Context) {
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

	var pipelines []models.Pipeline
	if err := config.DB.
		Where("company_id = ?", companyIDUUID).
		Preload("SourceObject.Connection.App").
		Preload("DestinationObject.Connection.App").
		Preload("Checkpoint").
		Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch pipelines",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pipelines": pipelines,
	})
}

// GetPipeline returns a specific pipeline with status
func (h *PipelineHandler) GetPipeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline ID",
		})
		return
	}

	var pipeline models.Pipeline
	if err := config.DB.
		Preload("Company").
		Preload("SourceObject.Connection.App").
		Preload("DestinationObject.Connection.App").
		Preload("Checkpoint").
		First(&pipeline, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Pipeline not found",
		})
		return
	}

	// Get pipeline status
	status, err := h.pipelineService.GetPipelineStatus(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get pipeline status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pipeline": pipeline,
		"status":   status,
	})
}

// CreatePipelineRequest represents the request body for creating a pipeline
type CreatePipelineRequest struct {
	CompanyID           uuid.UUID              `json:"company_id" binding:"required"`
	SourceObjectID      uuid.UUID              `json:"source_object_id" binding:"required"`
	DestinationObjectID uuid.UUID              `json:"destination_object_id" binding:"required"`
	SyncType            string                 `json:"sync_type"` // 'PULL', 'PUSH', 'BIDIRECTIONAL'
	ScheduleInterval    int                    `json:"schedule_interval"` // minutes
	FieldMapping        map[string]interface{} `json:"field_mapping"`
}

// CreatePipeline creates a new pipeline
func (h *PipelineHandler) CreatePipeline(c *gin.Context) {
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

	var req CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline data",
			"details": err.Error(),
		})
		return
	}

	// Use company_id from auth context instead of request body
	req.CompanyID = companyIDUUID

	// Validate source and destination objects exist and belong to the company
	var sourceObject models.DataObject
	if err := config.DB.Preload("Connection").First(&sourceObject, req.SourceObjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Source object not found",
		})
		return
	}

	// Verify source object belongs to the company
	if sourceObject.Connection.CompanyID != companyIDUUID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Source object does not belong to your company",
		})
		return
	}

	var destObject models.DataObject
	if err := config.DB.Preload("Connection").First(&destObject, req.DestinationObjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Destination object not found",
		})
		return
	}

	// Verify destination object belongs to the company
	if destObject.Connection.CompanyID != companyIDUUID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Destination object does not belong to your company",
		})
		return
	}

	// Set defaults
	if req.SyncType == "" {
		req.SyncType = "PULL"
	}
	if req.ScheduleInterval == 0 {
		req.ScheduleInterval = 60 // Default to 60 minutes
	}

	// Marshal field mapping
	fieldMappingJSON := "{}"
	if req.FieldMapping != nil {
		fieldMappingBytes, err := json.Marshal(req.FieldMapping)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid field mapping",
				"details": err.Error(),
			})
			return
		}
		fieldMappingJSON = string(fieldMappingBytes)
	}

	// Create pipeline
	pipeline := models.Pipeline{
		CompanyID:           req.CompanyID,
		SourceObjectID:      req.SourceObjectID,
		DestinationObjectID: req.DestinationObjectID,
		SyncType:            req.SyncType,
		ScheduleInterval:    req.ScheduleInterval,
		Status:              "ACTIVE",
		FieldMapping:        fieldMappingJSON,
	}

	if err := config.DB.Create(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create pipeline",
			"details": err.Error(),
		})
		return
	}

	// Load relations for response
	config.DB.
		Preload("SourceObject.Connection.App").
		Preload("DestinationObject.Connection.App").
		First(&pipeline, pipeline.ID)

	c.JSON(http.StatusCreated, pipeline)
}

// UpdatePipeline updates an existing pipeline
func (h *PipelineHandler) UpdatePipeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline ID",
		})
		return
	}

	var pipeline models.Pipeline
	if err := config.DB.First(&pipeline, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Pipeline not found",
		})
		return
	}

	var req struct {
		SyncType         string                 `json:"sync_type"`
		ScheduleInterval int                    `json:"schedule_interval"`
		Status           string                 `json:"status"`
		FieldMapping     map[string]interface{} `json:"field_mapping"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline data",
			"details": err.Error(),
		})
		return
	}

	// Update fields
	if req.SyncType != "" {
		pipeline.SyncType = req.SyncType
	}
	if req.ScheduleInterval > 0 {
		pipeline.ScheduleInterval = req.ScheduleInterval
	}
	if req.Status != "" {
		pipeline.Status = req.Status
	}
	if req.FieldMapping != nil {
		fieldMappingBytes, _ := json.Marshal(req.FieldMapping)
		pipeline.FieldMapping = string(fieldMappingBytes)
	}

	if err := config.DB.Save(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update pipeline",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// DeletePipeline deletes a pipeline (soft delete)
func (h *PipelineHandler) DeletePipeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline ID",
		})
		return
	}

	if err := config.DB.Delete(&models.Pipeline{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete pipeline",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pipeline deleted successfully",
	})
}

// ExecutePipeline manually triggers a pipeline execution
func (h *PipelineHandler) ExecutePipeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline ID",
		})
		return
	}

	// Execute pipeline in background
	go func() {
		if err := h.pipelineService.ExecutePipeline(id); err != nil {
			// Log error (in production, use proper logging)
			fmt.Printf("Pipeline execution failed: %v\n", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Pipeline execution started",
		"pipeline_id": id,
	})
}

// GetPipelineSyncRuns returns sync runs for a pipeline
func (h *PipelineHandler) GetPipelineSyncRuns(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid pipeline ID",
		})
		return
	}

	var syncRuns []models.SyncRun
	if err := config.DB.
		Where("pipeline_id = ?", id).
		Order("started_at DESC").
		Limit(100).
		Find(&syncRuns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch sync runs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sync_runs": syncRuns,
	})
}

