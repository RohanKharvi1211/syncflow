package controllers

import (
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CompanyHandler struct {
	access *ControllerAccess
}

func NewCompanyHandler() ControllerCompanyMethods {
	return &CompanyHandler{}
}

func NewCompanyHandlerWithAccess(access *ControllerAccess) ControllerCompanyMethods {
	return &CompanyHandler{access: access}
}

// GetCompanies returns all companies
func (h *CompanyHandler) GetCompanies(c *gin.Context) {
	var companies []models.Company
	if err := config.DB.Find(&companies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch companies",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": companies,
	})
}

// GetCompany returns a specific company
func (h *CompanyHandler) GetCompany(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company ID",
		})
		return
	}

	var company models.Company
	if err := config.DB.Preload("Users").Preload("Connections").Preload("Integrations").First(&company, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Company not found",
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// CreateCompanyRequest represents the request for creating a company with admin user
type CreateCompanyRequest struct {
	Name      string `json:"name" binding:"required"`
	Domain    string `json:"domain"`
	AdminEmail string `json:"admin_email" binding:"required"`
	AdminPassword string `json:"admin_password" binding:"required"`
	AdminFirstName string `json:"admin_first_name"`
	AdminLastName  string `json:"admin_last_name"`
}

// CreateCompany creates a new company with an admin user
func (h *CompanyHandler) CreateCompany(c *gin.Context) {
	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company data",
			"details": err.Error(),
		})
		return
	}

	// Start transaction
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create company
	company := models.Company{
		Name:     req.Name,
		Domain:   req.Domain,
		IsActive: true,
	}

	if err := tx.Create(&company).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create company",
			"details": err.Error(),
		})
		return
	}

	// Create admin user
	// TODO: Hash password properly
	adminUser := models.User{
		CompanyID: company.ID,
		Email:     req.AdminEmail,
		PasswordHash: req.AdminPassword, // TODO: Hash this properly
		FirstName: req.AdminFirstName,
		LastName:  req.AdminLastName,
		Role:      "admin",
		IsActive:  true,
	}

	if err := tx.Create(&adminUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create admin user",
			"details": err.Error(),
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save company",
			"details": err.Error(),
		})
		return
	}

	// Load relations
	config.DB.Preload("Users").First(&company, company.ID)

	c.JSON(http.StatusCreated, company)
}

// UpdateCompany updates an existing company
func (h *CompanyHandler) UpdateCompany(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company ID",
		})
		return
	}

	var company models.Company
	if err := config.DB.First(&company, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Company not found",
		})
		return
	}

	if err := c.ShouldBindJSON(&company); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company data",
			"details": err.Error(),
		})
		return
	}

	if err := config.DB.Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update company",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// DeleteCompany deletes a company (soft delete)
func (h *CompanyHandler) DeleteCompany(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company ID",
		})
		return
	}

	if err := config.DB.Delete(&models.Company{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete company",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Company deleted successfully",
	})
}
