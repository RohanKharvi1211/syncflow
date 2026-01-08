package controllers

import (
	"net/http"
	"strconv"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IntegrationSyncHandler struct {
	access      *ControllerAccess
	SyncService *services.IntegrationSyncService
}

func NewIntegrationSyncHandler() ControllerSyncMethods {
	return &IntegrationSyncHandler{
		SyncService: services.NewIntegrationSyncService(),
	}
}

func NewIntegrationSyncHandlerWithAccess(access *ControllerAccess) ControllerSyncMethods {
	return &IntegrationSyncHandler{
		access:      access,
		SyncService: services.NewIntegrationSyncService(),
	}
}

// SyncIntegration triggers a sync for a specific integration
func (h *IntegrationSyncHandler) SyncIntegration(c *gin.Context) {
	integrationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid integration ID",
		})
		return
	}

	triggerType := c.DefaultQuery("trigger_type", "manual")
	if triggerType != "manual" && triggerType != "scheduled" && triggerType != "webhook" {
		triggerType = "manual"
	}

	syncJob, err := h.SyncService.SyncIntegration(integrationID, triggerType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sync integration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, syncJob)
}

// GetSyncJobs returns sync jobs for an integration
func (h *IntegrationSyncHandler) GetSyncJobs(c *gin.Context) {
	integrationIDStr := c.Query("integration_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var jobs []models.SyncJob
	var total int64

	query := config.DB.Model(&models.SyncJob{}).Preload("Integration")

	if integrationIDStr != "" {
		integrationID, err := uuid.Parse(integrationIDStr)
		if err == nil {
			query = query.Where("integration_id = ?", integrationID)
		}
	}

	query.Count(&total)
	if err := query.Offset(offset).Limit(limit).Order("start_time DESC").Find(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch sync jobs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetSyncJob returns a specific sync job
func (h *IntegrationSyncHandler) GetSyncJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid sync job ID",
		})
		return
	}

	var job models.SyncJob
	if err := config.DB.Preload("Integration").Preload("SyncRecords").First(&job, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Sync job not found",
		})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetSyncRecords returns sync records with filtering
func (h *IntegrationSyncHandler) GetSyncRecords(c *gin.Context) {
	integrationIDStr := c.Query("integration_id")
	jobIDStr := c.Query("job_id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var records []models.SyncRecord
	var total int64

	query := config.DB.Model(&models.SyncRecord{}).Preload("Integration").Preload("Job")

	if integrationIDStr != "" {
		integrationID, err := uuid.Parse(integrationIDStr)
		if err == nil {
			query = query.Where("integration_id = ?", integrationID)
		}
	}

	if jobIDStr != "" {
		jobID, err := uuid.Parse(jobIDStr)
		if err == nil {
			query = query.Where("job_id = ?", jobID)
		}
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch sync records",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetSyncRecord returns a specific sync record
func (h *IntegrationSyncHandler) GetSyncRecord(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid sync record ID",
		})
		return
	}

	var record models.SyncRecord
	if err := config.DB.Preload("Integration").Preload("Job").First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Sync record not found",
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

// RetrySyncRecord retries a failed sync record
func (h *IntegrationSyncHandler) RetrySyncRecord(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid sync record ID",
		})
		return
	}

	if err := h.SyncService.RetrySyncRecord(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retry sync record",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync record retried successfully",
	})
}

// GetSyncStats returns sync statistics
func (h *IntegrationSyncHandler) GetSyncStats(c *gin.Context) {
	integrationIDStr := c.Query("integration_id")

	var stats struct {
		TotalRecords    int64 `json:"total_records"`
		SyncedRecords   int64 `json:"synced_records"`
		FailedRecords   int64 `json:"failed_records"`
		PendingRecords  int64 `json:"pending_records"`
	}

	query := config.DB.Model(&models.SyncRecord{})

	if integrationIDStr != "" {
		integrationID, err := uuid.Parse(integrationIDStr)
		if err == nil {
			query = query.Where("integration_id = ?", integrationID)
		}
	}

	query.Count(&stats.TotalRecords)
	query.Where("status = ?", "synced").Count(&stats.SyncedRecords)
	query.Where("status = ?", "failed").Count(&stats.FailedRecords)
	query.Where("status = ?", "pending_retry").Count(&stats.PendingRecords)

	c.JSON(http.StatusOK, stats)
}

