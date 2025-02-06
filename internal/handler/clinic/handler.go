package clinic

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/pkg/event"
	"github.com/jwalitptl/pkg/model"
	clinicService "github.com/jwalitptl/pkg/service/clinic"

	"github.com/jwalitptl/admin-api/internal/handler"
)

type Handler struct {
	service clinicService.Service
}

func NewHandler(service clinicService.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	clinics := r.Group("/clinics")
	{
		clinics.POST("", h.CreateClinic)
		clinics.GET("", h.ListClinics)
		clinics.GET("/:id", h.GetClinic)
		clinics.PUT("/:id", h.UpdateClinic)
		clinics.DELETE("/:id", h.DeleteClinic)
	}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	clinics := r.Group("/clinics")
	{
		clinics.POST("", eventTracker.TrackEvent("clinic", "create"), h.CreateClinic)
		clinics.PUT("/:id", eventTracker.TrackEvent("clinic", "update"), h.UpdateClinic)
		clinics.DELETE("/:id", eventTracker.TrackEvent("clinic", "delete"), h.DeleteClinic)
		clinics.GET("", h.ListClinics)
		clinics.GET("/:id", h.GetClinic)
	}
}

type createClinicRequest struct {
	OrganizationID string `json:"organization_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Location       string `json:"location" binding:"required"`
}

func (h *Handler) CreateClinic(c *gin.Context) {
	var req createClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	clinic := &model.Clinic{
		OrganizationID: orgID,
		Name:           req.Name,
		Location:       req.Location,
		Status:         "active",
	}

	if err := h.service.CreateClinic(c.Request.Context(), clinic); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = clinic
		ctx.Additional = map[string]interface{}{
			"organization_id": orgID,
		}
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(clinic))
}

func (h *Handler) GetClinic(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	clinic, err := h.service.GetClinic(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinic))
}

type updateClinicRequest struct {
	Name     string `json:"name" binding:"required"`
	Location string `json:"location" binding:"required"`
	Status   string `json:"status" binding:"required"`
}

func (h *Handler) UpdateClinic(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	var req updateClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	oldClinic, err := h.service.GetClinic(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	clinic := &model.Clinic{
		Base:     model.Base{ID: id},
		Name:     req.Name,
		Location: req.Location,
		Status:   req.Status,
	}

	if err := h.service.UpdateClinic(c.Request.Context(), clinic); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = oldClinic
		ctx.NewData = clinic
		ctx.Additional = map[string]interface{}{
			"clinic_id": id,
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinic))
}

func (h *Handler) DeleteClinic(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	clinic, err := h.service.GetClinic(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	if err := h.service.DeleteClinic(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = clinic
		ctx.Additional = map[string]interface{}{
			"clinic_id":       id,
			"organization_id": clinic.OrganizationID,
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListClinics(c *gin.Context) {
	orgID := c.Query("organization_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("organization_id is required"))
		return
	}

	organizationID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	clinics, err := h.service.ListClinics(c.Request.Context(), organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinics))
}
