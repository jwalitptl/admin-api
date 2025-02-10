package patient

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/internal/service/patient"
	"github.com/jwalitptl/admin-api/internal/service/region"
	"github.com/jwalitptl/admin-api/pkg/event"
)

type Handler struct {
	service              patient.Service
	outboxRepo           postgres.OutboxRepository
	*handler.BaseHandler // Embed BaseHandler for region functionality
}

func NewHandler(service patient.Service, outboxRepo postgres.OutboxRepository, regionSvc *region.Service) *Handler {
	return &Handler{
		service:    service,
		outboxRepo: outboxRepo,
		BaseHandler: &handler.BaseHandler{
			RegionSvc:     regionSvc,
			DefaultConfig: regionSvc.GetDefaultConfig(),
		},
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	patients := r.Group("/patients")
	{
		patients.POST("", h.CreatePatient)
		patients.GET("", h.ListPatients)
		patients.GET("/:id", h.GetPatient)
		patients.PUT("/:id", h.UpdatePatient)
		patients.DELETE("/:id", h.DeletePatient)

		patients.POST("/:id/records", h.AddMedicalRecord)
		patients.GET("/:id/records", h.ListMedicalRecords)
		patients.GET("/:id/records/:recordId", h.GetMedicalRecord)

		patients.POST("/:id/appointments", h.CreateAppointment)
		patients.GET("/:id/appointments", h.ListAppointments)
		patients.PUT("/:id/appointments/:appointmentId", h.UpdateAppointment)
		patients.DELETE("/:id/appointments/:appointmentId", h.CancelAppointment)

		patients.PUT("/:id/insurance", h.UpdateInsurance)
		patients.GET("/:id/insurance", h.GetInsurance)
	}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	patients := r.Group("/patients")
	{
		patients.POST("", eventTracker.TrackEvent("PATIENT", "CREATE"), h.CreatePatient)
		patients.PUT("/:id", eventTracker.TrackEvent("PATIENT", "UPDATE"), h.UpdatePatient)
		patients.DELETE("/:id", eventTracker.TrackEvent("PATIENT", "DELETE"), h.DeletePatient)
		patients.GET("", h.ListPatients)
		patients.GET("/:id", h.GetPatient)
	}
}

func (h *Handler) CreatePatient(c *gin.Context) {
	regionConfig := h.GetRegionConfig(c)

	// Apply region-specific validation
	if err := h.ValidateRegionCompliance(c); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	// Use region-specific features
	if enabled := regionConfig.Features["advanced_patient_profile"]; enabled {
		// Handle advanced profile features
	}

	var req model.CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	clinicID, err := uuid.Parse(req.ClinicID)
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	patient := &model.Patient{
		Base: model.Base{
			ID: uuid.New(), // Generate new UUID
		},
		ClinicID:    clinicID,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		DateOfBirth: req.DOB,
		Phone:       req.Phone,
		Address:     req.Address,
		Status:      req.Status,
	}

	patient, err = h.service.CreatePatient(c.Request.Context(), patient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Create outbox event
	payload, err := json.Marshal(patient)
	if err != nil {
		log.Printf("failed to marshal patient for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "PATIENT_CREATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(patient))
}

func (h *Handler) GetPatient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	patient, err := h.service.GetPatient(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": patient})
}

func (h *Handler) UpdatePatient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	var req model.UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patient := &model.Patient{
		Base:        model.Base{ID: id},
		FirstName:   *req.FirstName,
		LastName:    *req.LastName,
		Email:       *req.Email,
		DateOfBirth: *req.DateOfBirth,
		Phone:       *req.Phone,
		Address:     *req.Address,
		Status:      *req.Status,
	}

	if err := h.service.UpdatePatient(c.Request.Context(), patient); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create outbox event for update
	payload, err := json.Marshal(patient)
	if err != nil {
		log.Printf("Failed to marshal patient for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "PATIENT_UPDATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": patient})
}

func (h *Handler) DeletePatient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	if err := h.service.DeletePatient(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create outbox event for delete
	payload, err := json.Marshal(map[string]interface{}{"id": id})
	if err != nil {
		log.Printf("Failed to marshal patient ID for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "PATIENT_DELETE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "patient deleted successfully"})
}

func (h *Handler) ListPatients(c *gin.Context) {
	filters := make(map[string]interface{})

	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	patients, err := h.service.ListPatients(c.Request.Context(), &model.PatientFilters{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": patients})
}

func (h *Handler) AddMedicalRecord(c *gin.Context) {
	// Implementation of AddMedicalRecord
}

func (h *Handler) ListMedicalRecords(c *gin.Context) {
	// Implementation of ListMedicalRecords
}

func (h *Handler) GetMedicalRecord(c *gin.Context) {
	// Implementation of GetMedicalRecord
}

func (h *Handler) CreateAppointment(c *gin.Context) {
	var req model.CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	req.PatientID = patientID
	appointment, err := h.service.CreateAppointment(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, appointment)
}

func (h *Handler) ListAppointments(c *gin.Context) {
	// Implementation of ListAppointments
}

func (h *Handler) UpdateAppointment(c *gin.Context) {
	// Implementation of UpdateAppointment
}

func (h *Handler) CancelAppointment(c *gin.Context) {
	// Implementation of CancelAppointment
}

func (h *Handler) UpdateInsurance(c *gin.Context) {
	// Implementation of UpdateInsurance
}

func (h *Handler) GetInsurance(c *gin.Context) {
	// Implementation of GetInsurance
}
