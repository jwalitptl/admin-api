package clinic

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	clinicService "github.com/jwalitptl/admin-api/internal/service/clinic"
	"github.com/jwalitptl/admin-api/pkg/event"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type Handler struct {
	service    clinicService.ClinicServicer
	outboxRepo repository.OutboxRepository
}

func NewHandler(service clinicService.ClinicServicer, outboxRepo repository.OutboxRepository) *Handler {
	return &Handler{service: service, outboxRepo: outboxRepo}
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
		clinics.POST("", eventTracker.TrackEvent("CLINIC", "CREATE"), h.CreateClinic)
		clinics.PUT("/:id", eventTracker.TrackEvent("CLINIC", "UPDATE"), h.UpdateClinic)
		clinics.DELETE("/:id", eventTracker.TrackEvent("CLINIC", "DELETE"), h.DeleteClinic)
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

	// Create outbox event
	payload, err := json.Marshal(clinic)
	if err != nil {
		log.Printf("Failed to marshal clinic for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "CLINIC_CREATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
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

	_, err = h.service.GetClinic(c.Request.Context(), id)
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

	// Create outbox event for update
	payload, err := json.Marshal(clinic)
	if err != nil {
		log.Printf("Failed to marshal clinic for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "CLINIC_UPDATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
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

	_, err = h.service.GetClinic(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	if err := h.service.DeleteClinic(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Create outbox event for delete
	payload, err := json.Marshal(map[string]interface{}{"id": id})
	if err != nil {
		log.Printf("Failed to marshal clinic ID for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "CLINIC_DELETE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
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
