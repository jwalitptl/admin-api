package patient

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/pkg/event"
	"github.com/jwalitptl/pkg/model"
	"github.com/jwalitptl/pkg/service/patient"
)

type Handler struct {
	patientService patient.PatientService
}

func NewHandler(patientService patient.PatientService) *Handler {
	return &Handler{
		patientService: patientService,
	}
}

func (h *Handler) CreatePatient(c *gin.Context) {
	var req model.CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	clinicID, err := uuid.Parse(req.ClinicID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinic ID"})
		return
	}

	patient := &model.Patient{
		ClinicID: clinicID,
		Name:     req.Name,
		Email:    req.Email,
		Status:   req.Status,
	}

	if err := h.patientService.CreatePatient(c.Request.Context(), patient); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context for the middleware
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = patient
		ctx.Additional = map[string]interface{}{
			"clinic_id": clinicID,
		}
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": patient})
}

func (h *Handler) GetPatient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid patient ID"})
		return
	}

	patient, err := h.patientService.GetPatient(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": patient})
}

func (h *Handler) ListPatients(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Query("clinic_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinic ID"})
		return
	}

	patients, err := h.patientService.ListPatients(c.Request.Context(), clinicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": patients})
}

func (h *Handler) DeletePatient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid patient ID"})
		return
	}

	if err := h.patientService.DeletePatient(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context for the middleware
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = id
		ctx.Additional = map[string]interface{}{
			"patient_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) UpdatePatient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid patient ID"})
		return
	}

	oldPatient, err := h.patientService.GetPatient(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	var req model.UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	updatedPatient, err := h.patientService.UpdatePatient(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context for the middleware
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = oldPatient
		ctx.NewData = updatedPatient
		ctx.Additional = map[string]interface{}{
			"patient_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": updatedPatient})
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	patients := r.Group("/patients")
	{
		patients.POST("", eventTracker.TrackEvent("patient", "create"), h.CreatePatient)
		patients.PUT("/:id", eventTracker.TrackEvent("patient", "update"), h.UpdatePatient)
		patients.DELETE("/:id", eventTracker.TrackEvent("patient", "delete"), h.DeletePatient)
		patients.GET("", h.ListPatients)
		patients.GET("/:id", h.GetPatient)
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	patients := r.Group("/patients")
	{
		patients.POST("", h.CreatePatient)
		patients.PUT("/:id", h.UpdatePatient)
		patients.DELETE("/:id", h.DeletePatient)
		patients.GET("", h.ListPatients)
		patients.GET("/:id", h.GetPatient)
	}
}
