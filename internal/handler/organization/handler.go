package organization

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/model"
	accountService "github.com/jwalitptl/admin-api/internal/service/account"
	"github.com/jwalitptl/admin-api/pkg/event"
)

type Handler struct {
	service accountService.AccountServicer
}

func NewHandler(service accountService.AccountServicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	orgs := r.Group("/organizations")
	{
		orgs.POST("", eventTracker.TrackEvent("organization", "create"), h.CreateOrganization)
		orgs.GET("/:id", h.GetOrganization)
		orgs.PUT("/:id", eventTracker.TrackEvent("organization", "update"), h.UpdateOrganization)
		orgs.DELETE("/:id", eventTracker.TrackEvent("organization", "delete"), h.DeleteOrganization)
		orgs.GET("", h.ListOrganizations)
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	orgs := r.Group("/organizations")
	{
		orgs.POST("", h.CreateOrganization)
		orgs.GET("/:id", h.GetOrganization)
		orgs.PUT("/:id", h.UpdateOrganization)
		orgs.DELETE("/:id", h.DeleteOrganization)
		orgs.GET("", h.ListOrganizations)
	}
}

func (h *Handler) CreateOrganization(c *gin.Context) {
	var org model.Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	if err := h.service.CreateOrganization(c.Request.Context(), &org); err != nil {
		log.Printf("Failed to create organization: %v", err)
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(org))
}

func (h *Handler) GetOrganization(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	org, err := h.service.GetOrganization(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(org))
}

func (h *Handler) UpdateOrganization(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	var org model.Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}
	org.ID = id

	if err := h.service.UpdateOrganization(c.Request.Context(), &org); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) DeleteOrganization(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	if err := h.service.DeleteOrganization(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListOrganizations(c *gin.Context) {
	accountID, err := uuid.Parse(c.Query("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	orgs, err := h.service.ListOrganizations(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(orgs))
}
