package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BotProtection middleware blocks common bot/scanner paths
// This reduces log noise and prevents unnecessary resource usage
func BotProtection() gin.HandlerFunc {
	// Common scanner/bot paths that should be blocked
	blockedPaths := []string{
		"/config",
		"/.env",
		"/.git",
		"/wp-admin",
		"/wp-login",
		"/phpmyadmin",
		"/admin",
		"/administrator",
		"/.well-known",
		"/.git/config",
		"/config/keys",
		"/phpinfo",
		"/shell",
		"/cgi-bin",
		"/index.php",
		"/.DS_Store",
		"/robots.txt", // Optional: uncomment if you don't want to serve robots.txt
	}

	return func(c *gin.Context) {
		path := strings.ToLower(c.Request.URL.Path)

		// Allow all API paths to pass through (don't block legitimate API endpoints)
		if strings.HasPrefix(path, "/api/") || path == "/health" || path == "/" {
			c.Next()
			return
		}

		// Check if path matches any blocked pattern
		for _, blocked := range blockedPaths {
			if strings.HasPrefix(path, blocked) {
				// Return 404 immediately without processing or logging
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		}

		// Also block paths with suspicious file extensions (only non-API paths)
		suspiciousExtensions := []string{".php", ".jsp", ".asp", ".aspx", ".sh", ".bat", ".exe"}
		for _, ext := range suspiciousExtensions {
			if strings.HasSuffix(path, ext) {
				// Return 404 immediately without processing or logging
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		}

		c.Next()
	}
}

