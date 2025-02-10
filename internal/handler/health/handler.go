package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	db *sqlx.DB
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{
		db: db,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	health := r.Group("/health")
	{
		health.GET("/live", h.LivenessCheck)
		health.GET("/ready", h.ReadinessCheck)
	}
}

func (h *Handler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func (h *Handler) ReadinessCheck(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "DOWN",
			"reason": "Database connection failed",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
