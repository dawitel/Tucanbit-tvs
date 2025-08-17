package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "tvs",
		"version":   "1.0.0",
		"timestamp": time.Now(),
	})
}

// Ready returns readiness status
func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"service":   "tvs",
		"version":   "1.0.0",
		"timestamp": time.Now(),
	})
}
