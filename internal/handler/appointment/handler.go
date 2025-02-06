package appointment

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/pkg/event"
	"github.com/jwalitptl/pkg/model"
	"github.com/jwalitptl/pkg/service/appointment"
)

type Handler struct {
	service *appointment.Service
}

func NewHandler(service *appointment.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateAppointment(c *gin.Context) {
	var req model.CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	appointment, err := h.service.CreateAppointment(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = appointment
		ctx.Additional = map[string]interface{}{
			"patient_id": appointment.PatientID,
			"clinic_id":  appointment.ClinicID,
		}
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": appointment})
}

func (h *Handler) GetAppointment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid appointment ID"})
		return
	}

	appointment, err := h.service.GetAppointment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": appointment})
}

func (h *Handler) ListAppointments(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Query("clinic_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinic ID"})
		return
	}

	filters := make(map[string]interface{})

	// Add optional filters
	if id := c.Query("clinician_id"); id != "" {
		clinicianID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinician ID"})
			return
		}
		filters["clinician_id"] = clinicianID
	}

	if id := c.Query("patient_id"); id != "" {
		patientID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid patient ID"})
			return
		}
		filters["patient_id"] = patientID
	}

	if status := c.Query("status"); status != "" {
		filters["status"] = model.AppointmentStatus(status)
	}

	if date := c.Query("start_date"); date != "" {
		filters["start_date"] = date
	}

	if date := c.Query("end_date"); date != "" {
		filters["end_date"] = date
	}

	appointments, err := h.service.ListAppointments(c.Request.Context(), clinicID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": appointments})
}

func (h *Handler) UpdateAppointment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid appointment ID"})
		return
	}

	var req model.UpdateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	oldAppointment, err := h.service.GetAppointment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	appointment, err := h.service.UpdateAppointment(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = oldAppointment
		ctx.NewData = appointment
		ctx.Additional = map[string]interface{}{
			"appointment_id": id,
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": appointment})
}

func (h *Handler) DeleteAppointment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid appointment ID"})
		return
	}

	appointment, err := h.service.GetAppointment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	if err := h.service.DeleteAppointment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = appointment
		ctx.Additional = map[string]interface{}{
			"appointment_id": id,
			"patient_id":     appointment.PatientID,
			"clinic_id":      appointment.ClinicID,
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	appointments := r.Group("/appointments")
	{
		appointments.GET("/availability", h.GetClinicianAvailability)
		appointments.POST("", h.CreateAppointment)
		appointments.GET("", h.ListAppointments)
		appointments.GET("/:id", h.GetAppointment)
		appointments.PUT("/:id", h.UpdateAppointment)
		appointments.DELETE("/:id", h.DeleteAppointment)
	}
}

func (h *Handler) GetClinicianAvailability(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Query("clinician_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinician ID"})
		return
	}

	dateStr := c.Query("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid date format"})
		return
	}

	slots, err := h.service.GetClinicianAvailability(c.Request.Context(), clinicianID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": slots})
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	appointments := r.Group("/appointments")
	{
		appointments.POST("", eventTracker.TrackEvent("appointment", "create"), h.CreateAppointment)
		appointments.PUT("/:id", eventTracker.TrackEvent("appointment", "update"), h.UpdateAppointment)
		appointments.DELETE("/:id", eventTracker.TrackEvent("appointment", "delete"), h.DeleteAppointment)
		appointments.GET("", h.ListAppointments)
		appointments.GET("/:id", h.GetAppointment)
	}
}
