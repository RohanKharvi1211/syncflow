package controllers

import (
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	access *ControllerAccess
}

func NewUserHandler() ControllerUserMethods {
	return &UserHandler{}
}

func NewUserHandlerWithAccess(access *ControllerAccess) ControllerUserMethods {
	return &UserHandler{access: access}
}

// GetUsers returns all users for a company or finds a user by email
func (h *UserHandler) GetUsers(c *gin.Context) {
	utils.DebugRequest("GET", c.Request.URL.Path, map[string]interface{}{
		"email":      c.Query("email"),
		"company_id": c.Query("company_id"),
	})

	// Check if searching by email (for OAuth login)
	email := c.Query("email")
	if email != "" {
		utils.Debug("Searching for user by email: %s", email)
		var user models.User
		if err := config.DB.Where("email = ?", email).Preload("Company").First(&user).Error; err != nil {
			utils.Debug("User not found: %v", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		utils.DebugVar("Found user", map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"company_id": user.CompanyID,
		})

		// Return user with company_name for frontend compatibility
		responseMap := map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"company_id": user.CompanyID,
			"role":       user.Role,
		}

		// Add company name if company is loaded
		if user.Company.ID != uuid.Nil {
			responseMap["company_name"] = user.Company.Name
		}

		c.JSON(http.StatusOK, gin.H{
			"user": responseMap,
		})
		return
	}

	// Otherwise, get users by company_id
	companyIDStr := c.Query("company_id")
	if companyIDStr == "" {
		utils.Debug("Missing company_id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "company_id or email is required",
		})
		return
	}

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		utils.Debug("Invalid company_id format: %s", companyIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid company_id",
		})
		return
	}

	utils.Debug("Fetching users for company_id: %s", companyID)
	var users []models.User
	if err := config.DB.Where("company_id = ?", companyID).Preload("Company").Find(&users).Error; err != nil {
		utils.Debug("Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch users",
			"details": err.Error(),
		})
		return
	}
	utils.Debug("Found %d users for company %s", len(users), companyID)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

// GetUser returns a specific user
func (h *UserHandler) GetUser(c *gin.Context) {
	utils.DebugRequest("GET", c.Request.URL.Path, map[string]interface{}{
		"id": c.Param("id"),
	})

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.Debug("Invalid user ID format: %s", c.Param("id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	utils.Debug("Fetching user with ID: %s", id)
	var user models.User
	if err := config.DB.Preload("Company").First(&user, id).Error; err != nil {
		utils.Debug("User not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	utils.DebugVar("User found", user)
	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *gin.Context) {
	utils.DebugRequest("POST", c.Request.URL.Path, nil)

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		utils.Debug("Invalid JSON binding: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user data",
			"details": err.Error(),
		})
		return
	}

	utils.DebugVar("Creating user", user)
	if err := config.DB.Create(&user).Error; err != nil {
		utils.Debug("Database error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
		return
	}

	utils.Debug("User created successfully with ID: %s", user.ID)
	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user data",
			"details": err.Error(),
		})
		return
	}

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user (soft delete)
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	if err := config.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}
