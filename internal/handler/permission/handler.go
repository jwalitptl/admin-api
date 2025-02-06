package permission

import (
	"net/http"

	"github.com/jwalitptl/pkg/event"
	permissionService "github.com/jwalitptl/pkg/service/permission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *permissionService.Service
}

func NewHandler(service *permissionService.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": permissions})
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	perms := r.Group("/permissions")
	{
		perms.POST("", eventTracker.TrackEvent("permission", "create"), h.CreatePermission)
		perms.PUT("/:id", eventTracker.TrackEvent("permission", "update"), h.UpdatePermission)
		perms.DELETE("/:id", eventTracker.TrackEvent("permission", "delete"), h.DeletePermission)
		// Non-tracked endpoints
		perms.GET("", h.ListPermissions)
		perms.GET("/:id", h.GetPermission)
	}
}

func (h *Handler) CreatePermission(c *gin.Context) {
	// ... existing validation code ...

	if err := h.service.CreatePermission(c.Request.Context(), permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = permission
	}

	c.JSON(http.StatusCreated, gin.H{"data": permission})
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	// ... existing validation code ...

	oldPermission, err := h.service.GetPermission(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdatePermission(c.Request.Context(), permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = oldPermission
		ctx.NewData = permission
		ctx.Additional = map[string]interface{}{
			"permission_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": permission})
}

func (h *Handler) DeletePermission(c *gin.Context) {
	// ... existing validation code ...

	permission, err := h.service.GetPermission(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeletePermission(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = permission
		ctx.Additional = map[string]interface{}{
			"permission_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

func (h *Handler) GetPermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permission, err := h.service.GetPermission(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": permission})
}
