package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"syncflow-backend/internal/services"
	"syncflow-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GoogleSheetsHandler struct{}

func NewGoogleSheetsHandler() *GoogleSheetsHandler {
	return &GoogleSheetsHandler{}
}

// ListSheets lists all Google Sheets for a connection
func (h *GoogleSheetsHandler) ListSheets(c *gin.Context) {
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

	// Get connection_id from query
	connectionIDStr := c.Query("connection_id")
	if connectionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "connection_id is required",
		})
		return
	}

	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection_id",
		})
		return
	}

	// Verify connection belongs to user's company
	var connection models.Connection
	// Try to load with App relation first (if app_id exists)
	query := config.DB.Where("id = ? AND company_id = ?", connectionID, companyIDUUID)
	
	// Try with Preload first, if it fails, try without
	if err := query.Preload("App").First(&connection).Error; err != nil {
		// If Preload fails, try without it (old schema)
		if err := config.DB.Where("id = ? AND company_id = ?", connectionID, companyIDUUID).First(&connection).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Connection not found",
			})
			return
		}
	}

	// Verify it's a Google Sheets/Drive connection
	// Try to check App relation first (if it was loaded)
	if connection.App.ID != uuid.Nil {
		if connection.App.Name != "googlesheet" && connection.App.Name != "googledrive" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Connection is not a Google Sheets/Drive connection",
			})
			return
		}
	}
	// If App relation not loaded, we'll proceed (connection exists and belongs to company)
	
	// Refresh token if needed before using it
	if err := utils.RefreshTokenIfNeeded(&connection); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token refresh failed",
			"details": err.Error(),
		})
		return
	}

	// Reload connection to get updated token
	if err := config.DB.First(&connection, connection.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found",
		})
		return
	}

	// Additional validation: ensure we have access token
	if connection.AccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Connection does not have access token",
		})
		return
	}

	// Initialize Google Drive service
	driveService := services.NewGoogleDriveService()
	driveService.Authenticate(connection.AccessToken, connection.RefreshToken)

	// List Google Sheets
	sheets, err := driveService.ListGoogleSheets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list Google Sheets",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sheets": sheets,
	})
}

// GetSheetHeaders fetches headers from a Google Sheet
func (h *GoogleSheetsHandler) GetSheetHeaders(c *gin.Context) {
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

	// Get data_object_id from query
	dataObjectIDStr := c.Query("data_object_id")
	if dataObjectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "data_object_id is required",
		})
		return
	}

	dataObjectID, err := uuid.Parse(dataObjectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid data_object_id",
		})
		return
	}

	// Get data object and verify it belongs to user's company
	var dataObject models.DataObject
	if err := config.DB.Preload("Connection").Preload("Connection.App").First(&dataObject, dataObjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Data object not found",
			"details": err.Error(),
		})
		return
	}

	// Verify connection belongs to user's company
	if dataObject.Connection.CompanyID != companyIDUUID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Check if connection has access token
	if dataObject.Connection.AccessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Connection does not have an access token. Please reconnect.",
		})
		return
	}

	// Verify it's a Google Sheets connection
	if dataObject.Connection.AppID != uuid.Nil {
		var app models.App
		if err := config.DB.First(&app, dataObject.Connection.AppID).Error; err == nil {
			if app.Name != "googlesheet" && app.Name != "googledrive" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Data object is not a Google Sheet",
				})
				return
			}
		}
	}

	// Get spreadsheet ID from data object identifier
	spreadsheetID := dataObject.Identifier

	// Get range from query (optional, defaults to first row)
	// Use A1:Z1 format - Google Sheets API will use the first sheet by default
	range_ := c.Query("range")
	if range_ == "" {
		range_ = "A1:Z1" // Default to first row (first sheet)
	}

	// Create Google Sheets service and authenticate
	sheetsService := services.NewGoogleSheetsService()
	
	// Refresh token if needed before using it
	if err := utils.RefreshTokenIfNeeded(&dataObject.Connection); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token refresh failed",
			"details": err.Error(),
		})
		return
	}

	// Reload connection to get updated token
	if err := config.DB.Preload("Connection").First(&dataObject, dataObject.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Data object not found",
		})
		return
	}

	// Check if access token exists
	if dataObject.Connection.AccessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Connection does not have an access token. Please reconnect.",
		})
		return
	}
	
	sheetsService.Authenticate(dataObject.Connection.AccessToken, dataObject.Connection.RefreshToken)

	// Fetch headers
	headers, err := sheetsService.GetSheetHeaders(spreadsheetID, range_)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("ERROR: Failed to fetch sheet headers for spreadsheet %s: %v\n", spreadsheetID, err)
		
		// Check if it's a permission/API not enabled error
		errMsg := err.Error()
		if strings.Contains(errMsg, "API has not been used") || strings.Contains(errMsg, "it is disabled") || strings.Contains(errMsg, "PERMISSION_DENIED") {
			// Try to extract project ID from error message
			// Error format: "...project 485007484288..." or "...project=485007484288..."
			projectID := ""
			enableURL := "https://console.developers.google.com/apis/api/sheets.googleapis.com/overview"
			
			// Look for project ID in the error message
			if strings.Contains(errMsg, "project ") {
				// Try to find pattern like "project 485007484288" or "project=485007484288"
				parts := strings.Split(errMsg, "project")
				if len(parts) > 1 {
					// Extract the number after "project"
					projectPart := strings.TrimSpace(parts[1])
					// Find the first sequence of digits
					for i, char := range projectPart {
						if char >= '0' && char <= '9' {
							// Extract the number
							numStr := ""
							for j := i; j < len(projectPart) && projectPart[j] >= '0' && projectPart[j] <= '9'; j++ {
								numStr += string(projectPart[j])
							}
							if numStr != "" {
								projectID = numStr
								enableURL = fmt.Sprintf("https://console.developers.google.com/apis/api/sheets.googleapis.com/overview?project=%s", projectID)
								break
							}
						}
					}
				}
			}
			
			// If we found a project ID in the error URL, use it
			if strings.Contains(errMsg, "project=") {
				// Extract from URL pattern
				urlParts := strings.Split(errMsg, "project=")
				if len(urlParts) > 1 {
					projectPart := strings.Split(urlParts[1], " ")[0]
					projectPart = strings.Split(projectPart, "&")[0]
					projectPart = strings.Split(projectPart, "\"")[0]
					projectPart = strings.Split(projectPart, "'")[0]
					if projectPart != "" {
						projectID = projectPart
						enableURL = fmt.Sprintf("https://console.developers.google.com/apis/api/sheets.googleapis.com/overview?project=%s", projectID)
					}
				}
			}
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Google Sheets API Permission Required",
				"message": "The Google Sheets API needs to be enabled in your Google Cloud project to read sheet data.",
				"instructions": []string{
					"Click the button below to open Google Cloud Console",
					"Click 'Enable' on the Google Sheets API page",
					"Wait a few minutes for changes to propagate",
					"Refresh this page and try again",
				},
				"enable_url": enableURL,
				"help_url": "https://console.cloud.google.com/apis/library/sheets.googleapis.com",
				"project_id": projectID,
				"details": errMsg,
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch sheet headers",
			"details": errMsg,
			"spreadsheet_id": spreadsheetID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"headers": headers,
		"spreadsheet_id": spreadsheetID,
	})
}


