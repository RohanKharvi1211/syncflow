package controllers

import (
	"net/http"
	"strconv"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	SyncService *services.MultiTenantSyncService
}

func NewSyncHandler() *SyncHandler {
	return &SyncHandler{
		SyncService: services.NewMultiTenantSyncService(),
	}
}

// @Summary Trigger sync for specific company and record type
// @Description Manually trigger synchronization for a specific company and record type
// @Tags sync
// @Accept json
// @Produce json
// @Param company_id path int true "Company ID"
// @Param record_type path string true "Record Type (Account, Contact, Opportunity)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/{company_id}/{record_type} [post]
func (h *SyncHandler) TriggerSyncForCompany(c *gin.Context) {
	companyIDStr := c.Param("company_id")
	recordType := c.Param("record_type")
	
	companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company ID",
		})
		return
	}
	
	if err := h.SyncService.SyncRecordsForCompany(uint(companyID), recordType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sync records",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync triggered successfully",
		"company_id": companyID,
		"record_type": recordType,
	})
}

// @Summary Get sync records
// @Description Get paginated list of sync records
// @Tags sync
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Records per page" default(10)
// @Param company_id query int false "Filter by company ID"
// @Param status query string false "Filter by status"
// @Param record_type query string false "Filter by record type"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/records [get]
func (h *SyncHandler) GetSyncRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	companyIDStr := c.Query("company_id")
	status := c.Query("status")
	recordType := c.Query("record_type")

	offset := (page - 1) * limit

	var records []models.LegacySyncRecord
	var total int64

	query := config.DB.Model(&models.LegacySyncRecord{}).Preload("Company")

	if companyIDStr != "" {
		companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
		if err == nil {
			query = query.Where("company_id = ?", companyID)
		}
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if recordType != "" {
		query = query.Where("record_type = ?", recordType)
	}

	// Get total count
	query.Count(&total)

	// Get records with pagination
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

// @Summary Get sync record by ID
// @Description Get detailed information about a specific sync record
// @Tags sync
// @Accept json
// @Produce json
// @Param id path int true "Sync Record ID"
// @Success 200 {object} models.SyncRecord
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/records/{id} [get]
func (h *SyncHandler) GetSyncRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid record ID",
		})
		return
	}

	var record models.LegacySyncRecord
	if err := config.DB.Preload("Company").Preload("SyncLogs").First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Record not found",
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

// @Summary Get sync statistics
// @Description Get synchronization statistics and metrics
// @Tags sync
// @Accept json
// @Produce json
// @Param company_id query int false "Filter by company ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/stats [get]
func (h *SyncHandler) GetSyncStats(c *gin.Context) {
	companyIDStr := c.Query("company_id")
	
	var stats struct {
		TotalRecords     int64 `json:"total_records"`
		SyncedRecords    int64 `json:"synced_records"`
		FailedRecords    int64 `json:"failed_records"`
		PendingRecords   int64 `json:"pending_records"`
		RetryingRecords  int64 `json:"retrying_records"`
	}

	query := config.DB.Model(&models.LegacySyncRecord{})
	
	if companyIDStr != "" {
		companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
		if err == nil {
			query = query.Where("company_id = ?", companyID)
		}
	}

	// Get counts by status
	query.Count(&stats.TotalRecords)
	query.Where("status = ?", "synced").Count(&stats.SyncedRecords)
	query.Where("status = ?", "failed").Count(&stats.FailedRecords)
	query.Where("status = ?", "pending").Count(&stats.PendingRecords)
	query.Where("status = ?", "retrying").Count(&stats.RetryingRecords)

	c.JSON(http.StatusOK, stats)
}

// @Summary Retry failed syncs
// @Description Retry all failed synchronization attempts
// @Tags sync
// @Accept json
// @Produce json
// @Param company_id query int false "Filter by company ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/sync/retry [post]
func (h *SyncHandler) RetryFailedSyncs(c *gin.Context) {
	companyIDStr := c.Query("company_id")
	
	if companyIDStr != "" {
		companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid company ID",
			})
			return
		}
		
		// Retry for specific company
		if err := h.retryFailedSyncsForCompany(uint(companyID)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retry syncs",
				"details": err.Error(),
			})
			return
		}
	} else {
		// Retry for all companies
		if err := h.retryAllFailedSyncs(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retry syncs",
				"details": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Retry process completed",
	})
}

func (h *SyncHandler) retryFailedSyncsForCompany(companyID uint) error {
	var failedRecords []models.LegacySyncRecord
	result := config.DB.Where("company_id = ? AND status = ? AND retry_count < ?", 
		companyID, "failed", 3).Find(&failedRecords)
	
	if result.Error != nil {
		return result.Error
	}

	for _, record := range failedRecords {
		if err := h.SyncService.SyncRecordsForCompany(companyID, record.RecordType); err != nil {
			continue
		}
	}

	return nil
}

func (h *SyncHandler) retryAllFailedSyncs() error {
	var failedRecords []models.LegacySyncRecord
	result := config.DB.Where("status = ? AND retry_count < ?", "failed", 3).Find(&failedRecords)
	
	if result.Error != nil {
		return result.Error
	}

	// Group by company and record type
	companyRecordTypes := make(map[uint]map[string]bool)
	for _, record := range failedRecords {
		if companyRecordTypes[record.CompanyID] == nil {
			companyRecordTypes[record.CompanyID] = make(map[string]bool)
		}
		companyRecordTypes[record.CompanyID][record.RecordType] = true
	}

	// Retry for each company and record type combination
	for companyID, recordTypes := range companyRecordTypes {
		for recordType := range recordTypes {
			if err := h.SyncService.SyncRecordsForCompany(companyID, recordType); err != nil {
				continue
			}
		}
	}

	return nil
}
