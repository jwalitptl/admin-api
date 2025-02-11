package clinic

import (
	"context"
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
		clinics.POST("", eventTracker.TrackEvent("clinic", "create"), h.CreateClinic)
		clinics.PUT("/:id", eventTracker.TrackEvent("clinic", "update"), h.UpdateClinic)
		clinics.DELETE("/:id", eventTracker.TrackEvent("clinic", "delete"), h.DeleteClinic)
		clinics.POST("/:id/staff", eventTracker.TrackEvent("clinic", "staff_assign"), h.AssignStaff)
		clinics.GET("/:id/staff", h.ListStaff)
		clinics.DELETE("/:id/staff/:userId", eventTracker.TrackEvent("clinic", "staff_remove"), h.RemoveStaff)
		clinics.GET("", h.ListClinics)
		clinics.GET("/:id", h.GetClinic)
	}
}

type createClinicRequest struct {
	OrganizationID string `json:"organization_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Location       string `json:"location" binding:"required"`
	Status         string `json:"status" binding:"required,oneof=active inactive"`
	RegionCode     string `json:"region_code"`
}

type AssignStaffRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

func (h *Handler) CreateClinic(c *gin.Context) {
	var req createClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Debug - Validation error: %v", err)
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	log.Printf("Debug - Request received: %+v", req)

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		log.Printf("Debug - Organization ID parse error: %v", err)
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	clinic := &model.Clinic{
		OrganizationID: orgID,
		Name:           req.Name,
		Location:       req.Location,
		Status:         req.Status,
		RegionCode:     req.RegionCode,
	}

	log.Printf("Debug - Creating clinic: %+v", clinic)
	if err := h.service.CreateClinic(c.Request.Context(), clinic); err != nil {
		log.Printf("Debug - Creation error: %v", err)
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	log.Printf("Debug - Clinic created successfully: %+v", clinic)

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

	log.Printf("Debug - Sending response")
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

func (h *Handler) UpdateClinic(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	// Get existing clinic to preserve organization_id
	existingClinic, err := h.service.GetClinic(c.Request.Context(), clinicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	var clinic model.Clinic
	if err := c.ShouldBindJSON(&clinic); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	clinic.ID = clinicID
	clinic.OrganizationID = existingClinic.OrganizationID

	if err := h.service.UpdateClinic(c.Request.Context(), &clinic); err != nil {
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

	search := c.Query("search")
	status := c.Query("status")

	ctx := context.WithValue(c.Request.Context(), "search", search)
	ctx = context.WithValue(ctx, "status", status)

	clinics, err := h.service.ListClinics(ctx, organizationID, search, status)
	if err != nil {
		log.Printf("Debug - List clinics error: %v", err)
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinics))
}

func (h *Handler) AssignStaff(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	var req AssignStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid request body"))
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid user ID"))
		return
	}

	if err := h.service.AssignStaff(c.Request.Context(), clinicID, userID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListStaff(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	staff, err := h.service.ListStaff(c.Request.Context(), clinicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(staff))
}

func (h *Handler) RemoveStaff(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid user ID"))
		return
	}

	if err := h.service.RemoveStaff(c.Request.Context(), clinicID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}
