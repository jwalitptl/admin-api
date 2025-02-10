package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler contains dependencies for all handlers
type Handler struct {
	// Add any dependencies here when needed
}

// NewHandler creates a new handler instance
func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now(),
	})
}

func (h *Handler) ReadinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now(),
	})
}

func (h *Handler) MetricsHandler(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
