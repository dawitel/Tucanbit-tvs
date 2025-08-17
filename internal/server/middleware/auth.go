package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth middleware validates API key authentication
func APIKeyAuth(expectedAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}

		// Get API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Try Authorization header as fallback
			apiKey = c.GetHeader("Authorization")
			if apiKey != "" && len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:]
			}
		}

		// Validate API key
		if apiKey != expectedAPIKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid or missing API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
