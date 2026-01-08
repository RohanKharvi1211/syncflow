package controllers

import (
	"net/http"
	"strconv"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type FieldMappingHandler struct {
	FieldMappingService *services.FieldMappingService
}

func NewFieldMappingHandler() *FieldMappingHandler {
	return &FieldMappingHandler{
		FieldMappingService: services.NewFieldMappingService(),
	}
}

// @Summary Get field mappings for company
// @Description Get all field mappings for a specific company
// @Tags field-mapping
// @Accept json
// @Produce json
// @Param company_id query int true "Company ID"
// @Param source_system query string false "Source System"
// @Param target_system query string false "Target System"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/field-mappings [get]
func (h *FieldMappingHandler) GetFieldMappings(c *gin.Context) {
	companyIDStr := c.Query("company_id")
	if companyIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "company_id is required",
		})
		return
	}

	companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company_id",
		})
		return
	}

	sourceSystem := c.Query("source_system")
	targetSystem := c.Query("target_system")

	if sourceSystem != "" && targetSystem != "" {
		mappings, err := h.FieldMappingService.GetFieldMappings(uint(companyID), sourceSystem, targetSystem)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch field mappings",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"mappings": mappings,
		})
		return
	}

	// If no specific systems requested, return all mappings for company
	// This would require a different service method
	c.JSON(http.StatusOK, gin.H{
		"message": "Please specify both source_system and target_system",
	})
}

// @Summary Create field mapping
// @Description Create a new field mapping
// @Tags field-mapping
// @Accept json
// @Produce json
// @Param mapping body models.FieldMapping true "Field mapping data"
// @Success 201 {object} models.FieldMapping
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/field-mappings [post]
func (h *FieldMappingHandler) CreateFieldMapping(c *gin.Context) {
	var mapping models.FieldMapping
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid field mapping data",
			"details": err.Error(),
		})
		return
	}

	if err := h.FieldMappingService.CreateFieldMapping(&mapping); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create field mapping",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, mapping)
}

// @Summary Update field mapping
// @Description Update an existing field mapping
// @Tags field-mapping
// @Accept json
// @Produce json
// @Param id path int true "Field Mapping ID"
// @Param mapping body models.FieldMapping true "Field mapping data"
// @Success 200 {object} models.FieldMapping
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/field-mappings/{id} [put]
func (h *FieldMappingHandler) UpdateFieldMapping(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid field mapping ID",
		})
		return
	}

	var mapping models.FieldMapping
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid field mapping data",
			"details": err.Error(),
		})
		return
	}

	mapping.ID = uint(id)
	if err := h.FieldMappingService.UpdateFieldMapping(&mapping); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update field mapping",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// @Summary Delete field mapping
// @Description Delete a field mapping
// @Tags field-mapping
// @Accept json
// @Produce json
// @Param id path int true "Field Mapping ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/field-mappings/{id} [delete]
func (h *FieldMappingHandler) DeleteFieldMapping(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid field mapping ID",
		})
		return
	}

	if err := h.FieldMappingService.DeleteFieldMapping(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete field mapping",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Field mapping deleted successfully",
	})
}

// @Summary Test field mapping
// @Description Test a field mapping with sample data
// @Tags field-mapping
// @Accept json
// @Produce json
// @Param test_data body map[string]interface{} true "Test data and mapping configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/field-mappings/test [post]
func (h *FieldMappingHandler) TestFieldMapping(c *gin.Context) {
	var testData struct {
		CompanyID    uint                   `json:"company_id"`
		SourceSystem string                 `json:"source_system"`
		TargetSystem string                 `json:"target_system"`
		SourceObject string                 `json:"source_object"`
		TargetObject string                 `json:"target_object"`
		SourceData   map[string]interface{} `json:"source_data"`
	}

	if err := c.ShouldBindJSON(&testData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid test data",
			"details": err.Error(),
		})
		return
	}

	result, err := h.FieldMappingService.TransformData(
		testData.CompanyID,
		testData.SourceSystem,
		testData.TargetSystem,
		testData.SourceObject,
		testData.TargetObject,
		testData.SourceData,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Field mapping test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transformed_data": result,
		"original_data": testData.SourceData,
	})
}
