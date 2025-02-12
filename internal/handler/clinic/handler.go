package clinic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

		// Service routes
		clinics.POST("/:id/services", eventTracker.TrackEvent("service", "create"), h.CreateService)
		clinics.GET("/:id/services", h.ListServices)
		clinics.GET("/:id/services/:serviceId", h.GetService)
		clinics.PUT("/:id/services/:serviceId", eventTracker.TrackEvent("service", "update"), h.UpdateService)
		clinics.DELETE("/:id/services/:serviceId", eventTracker.TrackEvent("service", "delete"), h.DeleteService)
		clinics.PATCH("/:id/services/:serviceId/deactivate", eventTracker.TrackEvent("service", "deactivate"), h.DeactivateService)
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

	// First delete associated clinic_staff records
	if err := h.service.DeleteClinicStaff(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(fmt.Sprintf("failed to delete clinic staff: %v", err)))
		return
	}

	// Then delete the clinic
	if err := h.service.DeleteClinic(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(fmt.Sprintf("failed to delete clinic: %v", err)))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse("clinic deleted successfully"))
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

// Service handler methods
func (h *Handler) CreateService(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	var service model.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		log.Printf("Debug - Validation error: %v", err)
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	// Validate required fields
	if service.Name == "" {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("service name is required"))
		return
	}
	if service.Duration <= 0 {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("duration must be positive"))
		return
	}
	if service.Price < 0 {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("price cannot be negative"))
		return
	}

	service.ClinicID = clinicID
	if err := h.service.CreateService(c.Request.Context(), &service); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(service))
}

func (h *Handler) ListServices(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	search := c.Query("search")
	isActive := c.Query("is_active")

	services, err := h.service.ListServices(c.Request.Context(), clinicID, search, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(services))
}

func (h *Handler) GetService(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	serviceID, err := uuid.Parse(c.Param("serviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid service ID"))
		return
	}

	service, err := h.service.GetService(c.Request.Context(), clinicID, serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(service))
}

func (h *Handler) UpdateService(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	serviceID, err := uuid.Parse(c.Param("serviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid service ID"))
		return
	}

	var service model.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		log.Printf("Debug - Service update bind error: %v", err)
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	service.ClinicID = clinicID
	service.ID = serviceID
	service.UpdatedAt = time.Now()
	// Set default status if not provided
	if service.Status == "" {
		service.Status = "active"
	}

	log.Printf("Debug - Updating service: %+v", service)
	if err := h.service.UpdateService(c.Request.Context(), &service); err != nil {
		log.Printf("Debug - Service update error: %v", err)
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(service))
}

func (h *Handler) DeleteService(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	serviceID, err := uuid.Parse(c.Param("serviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid service ID"))
		return
	}

	if err := h.service.DeleteService(c.Request.Context(), clinicID, serviceID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(fmt.Sprintf("failed to delete service: %v", err)))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse("service deleted successfully"))
}

func (h *Handler) DeactivateService(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	serviceID, err := uuid.Parse(c.Param("serviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid service ID"))
		return
	}

	if err := h.service.DeactivateService(c.Request.Context(), clinicID, serviceID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}
