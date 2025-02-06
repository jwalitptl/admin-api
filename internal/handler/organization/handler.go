package organization

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jwalitptl/pkg/event"
	organizationService "github.com/jwalitptl/pkg/service/organization"
)

type Handler struct {
	service organizationService.Service
}

func NewHandler(service organizationService.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	orgs := r.Group("/organizations")
	{
		orgs.POST("", eventTracker.TrackEvent("organization", "create"), h.CreateOrganization)
		orgs.PUT("/:id", eventTracker.TrackEvent("organization", "update"), h.UpdateOrganization)
		orgs.DELETE("/:id", eventTracker.TrackEvent("organization", "delete"), h.DeleteOrganization)
		// Non-tracked endpoints
		orgs.GET("", h.ListOrganizations)
		orgs.GET("/:id", h.GetOrganization)
	}
}

func (h *Handler) CreateOrganization(c *gin.Context) {
	// ... existing validation code ...

	if err := h.service.CreateOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = org
	}

	c.JSON(http.StatusCreated, gin.H{"data": org})
}

func (h *Handler) UpdateOrganization(c *gin.Context) {
	// ... existing validation code ...

	oldOrg, err := h.service.GetOrganization(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = oldOrg
		ctx.NewData = org
		ctx.Additional = map[string]interface{}{
			"organization_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": org})
}

func (h *Handler) DeleteOrganization(c *gin.Context) {
	// ... existing validation code ...

	org, err := h.service.GetOrganization(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteOrganization(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = org
		ctx.Additional = map[string]interface{}{
			"organization_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted successfully"})
}
