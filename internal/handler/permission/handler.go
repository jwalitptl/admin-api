package permission

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
	permissionService "github.com/jwalitptl/admin-api/internal/service/permission"

	"github.com/gin-gonic/gin"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/pkg/event"
)

type Handler struct {
	service    *permissionService.Service
	outboxRepo postgres.OutboxRepository
}

func NewHandler(service *permissionService.Service, outboxRepo postgres.OutboxRepository) *Handler {
	return &Handler{
		service:    service,
		outboxRepo: outboxRepo,
	}
}

// RegisterRoutes registers all routes for the permission handler
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	permissions := group.Group("/permissions")
	{
		// Non-tracked endpoints
		permissions.GET("", h.ListPermissions)
		permissions.GET("/:id", h.GetPermission)

		// Event-tracked endpoints
		permissions.POST("", h.CreatePermission)
		permissions.PUT("/:id", h.UpdatePermission)
		permissions.DELETE("/:id", h.DeletePermission)
	}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	permissions := r.Group("/permissions")
	{
		permissions.POST("", eventTracker.TrackEvent("PERMISSION", "CREATE"), h.CreatePermission)
		permissions.PUT("/:id", eventTracker.TrackEvent("PERMISSION", "UPDATE"), h.UpdatePermission)
		permissions.DELETE("/:id", eventTracker.TrackEvent("PERMISSION", "DELETE"), h.DeletePermission)
		permissions.GET("", h.ListPermissions)
		permissions.GET("/:id", h.GetPermission)
	}
}

func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permissions)
}

func (h *Handler) CreatePermission(c *gin.Context) {
	var permission model.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreatePermission(c.Request.Context(), &permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create outbox event
	payload, err := json.Marshal(permission)
	if err != nil {
		log.Printf("Failed to marshal permission for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "PERMISSION_CREATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusCreated, permission)
}

func (h *Handler) GetPermission(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	permission, err := h.service.GetPermission(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permission)
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	var permission model.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePermission(c.Request.Context(), &permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create outbox event for update
	payload, err := json.Marshal(permission)
	if err != nil {
		log.Printf("Failed to marshal permission for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "PERMISSION_UPDATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusOK, permission)
}

func (h *Handler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.service.DeletePermission(c.Request.Context(), uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create outbox event for delete
	payload, err := json.Marshal(map[string]interface{}{"id": uid})
	if err != nil {
		log.Printf("Failed to marshal permission ID for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "PERMISSION_DELETE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
		}
	}

	c.Status(http.StatusNoContent)
}
