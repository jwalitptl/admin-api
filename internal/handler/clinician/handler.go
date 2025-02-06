package clinician

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/jwalitptl/pkg/event"
	"github.com/jwalitptl/pkg/model"
	"github.com/jwalitptl/pkg/security"
	"github.com/jwalitptl/pkg/service/clinician"

	"github.com/jwalitptl/admin-api/internal/handler"
)

type Handler struct {
	service clinician.Service
	db      *sqlx.DB
}

func NewHandler(service clinician.Service, db *sqlx.DB) *Handler {
	return &Handler{service: service, db: db}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	clinicians := r.Group("/clinicians")
	{
		clinicians.POST("", h.CreateClinician)
		clinicians.GET("", h.ListClinicians)
		clinicians.GET("/:id", h.GetClinician)
		clinicians.PUT("/:id", h.UpdateClinician)
		clinicians.DELETE("/:id", h.DeleteClinician)
		clinicians.PUT("/:id/password", h.UpdatePassword)

		// Clinic assignments
		clinicians.POST("/:id/clinics/:clinic_id", h.AssignToClinic)
		clinicians.DELETE("/:id/clinics/:clinic_id", h.RemoveFromClinic)
		clinicians.GET("/:id/clinics", h.ListClinicianClinics)

		// Role assignments
		clinicians.POST("/:id/roles/:role_id", h.AssignRole)
		clinicians.DELETE("/:id/roles/:role_id", h.RemoveRole)
		clinicians.GET("/:id/roles", h.ListClinicianRoles)
	}

	// Clinic-specific routes
	r.GET("/clinics/:id/clinicians", h.ListClinicClinicians)
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	clinicians := r.Group("/clinicians")
	{
		clinicians.POST("", eventTracker.TrackEvent("clinician", "create"), h.CreateClinician)
		clinicians.PUT("/:id", eventTracker.TrackEvent("clinician", "update"), h.UpdateClinician)
		clinicians.DELETE("/:id", eventTracker.TrackEvent("clinician", "delete"), h.DeleteClinician)
		clinicians.GET("", h.ListClinicians)
		clinicians.GET("/:id", h.GetClinician)
		clinicians.GET("/:id/clinics", h.ListClinicianClinics)
		clinicians.GET("/:id/roles", h.ListClinicianRoles)

		// Role assignments
		clinicians.POST("/:id/roles/:role_id", eventTracker.TrackEvent("clinician_role", "create"), h.AssignRole)
		clinicians.DELETE("/:id/roles/:role_id", eventTracker.TrackEvent("clinician_role", "delete"), h.RemoveRole)

		// Clinic assignments
		clinicians.POST("/:id/clinics/:clinic_id", eventTracker.TrackEvent("clinician_clinic", "create"), h.AssignToClinic)
		clinicians.DELETE("/:id/clinics/:clinic_id", eventTracker.TrackEvent("clinician_clinic", "delete"), h.RemoveFromClinic)
	}
}

func (h *Handler) CreateClinician(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	clinician := &model.Clinician{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
		Status:   "active",
	}

	if err := h.service.CreateClinician(c.Request.Context(), clinician); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		// Remove sensitive data before emitting event
		clinician.Password = ""
		clinician.PasswordHash = ""
		ctx.NewData = clinician
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": clinician})
}

func (h *Handler) GetClinician(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	clinician, err := h.service.GetClinician(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinician))
}

type updateClinicianRequest struct {
	Name   string `json:"name" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	Status string `json:"status" binding:"required"`
}

func (h *Handler) UpdateClinician(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	var req updateClinicianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	oldClinician, err := h.service.GetClinician(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	clinician := &model.Clinician{
		Base: model.Base{
			ID: id,
		},
		Name:   req.Name,
		Email:  req.Email,
		Status: req.Status,
	}

	if err := h.service.UpdateClinician(c.Request.Context(), clinician); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		// Remove sensitive data before emitting event
		oldClinician.PasswordHash = ""
		clinician.PasswordHash = ""
		ctx.OldData = oldClinician
		ctx.NewData = clinician
		ctx.Additional = map[string]interface{}{
			"clinician_id": id,
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinician))
}

func (h *Handler) DeleteClinician(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	clinician, err := h.service.GetClinician(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	if err := h.service.DeleteClinician(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		// Remove sensitive data before emitting event
		clinician.PasswordHash = ""
		ctx.OldData = clinician
		ctx.Additional = map[string]interface{}{
			"clinician_id": id,
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListClinicians(c *gin.Context) {
	clinicians, err := h.service.ListClinicians(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinicians))
}

func (h *Handler) AssignToClinic(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinician ID"})
		return
	}

	clinicID, err := uuid.Parse(c.Param("clinic_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinic ID"})
		return
	}

	// Debug logging
	fmt.Printf("Handler - Assigning clinician %s to clinic %s\n", clinicianID, clinicID)

	// Also verify the clinic exists directly
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM clinics WHERE id = $1)"
	err = h.db.QueryRowContext(c.Request.Context(), query, clinicID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	fmt.Printf("Handler - Clinic exists in DB: %v\n", exists)

	if err := h.service.AssignToClinic(c.Request.Context(), clinicianID, clinicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = map[string]interface{}{
			"clinician_id": clinicianID,
			"clinic_id":    clinicID,
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) RemoveFromClinic(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	clinicID, err := uuid.Parse(c.Param("clinic_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	if err := h.service.RemoveFromClinic(c.Request.Context(), clinicID, clinicianID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = map[string]interface{}{
			"clinician_id": clinicianID,
			"clinic_id":    clinicID,
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListClinicianClinics(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	clinics, err := h.service.ListClinicianClinics(c.Request.Context(), clinicianID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinics))
}

func (h *Handler) ListClinicClinicians(c *gin.Context) {
	clinicID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinic ID"))
		return
	}

	clinicians, err := h.service.ListClinicClinicians(c.Request.Context(), clinicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(clinicians))
}

// Add password update functionality
type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func (h *Handler) UpdatePassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	var req updatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	// Get current clinician
	clinician, err := h.service.GetClinician(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Verify current password
	if !security.CheckPassword(clinician.PasswordHash, req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("invalid current password"))
		return
	}

	// Hash new password
	hashedPassword, err := security.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse("failed to process password"))
		return
	}

	// Update password
	clinician.PasswordHash = hashedPassword
	if err := h.service.UpdateClinician(c.Request.Context(), clinician); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) AssignRole(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid clinician ID"})
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid role ID"})
		return
	}

	// Get the role to find its organization
	role, err := h.service.GetRole(c.Request.Context(), roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	if role.OrganizationID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "role must belong to an organization"})
		return
	}

	if err := h.service.AssignRoleToClinician(c.Request.Context(), clinicianID, roleID, *role.OrganizationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.NewData = map[string]interface{}{
			"clinician_id": clinicianID,
			"role_id":      roleID,
		}
		ctx.Additional = map[string]interface{}{
			"organization_id": *role.OrganizationID,
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) RemoveRole(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	if err := h.service.RemoveRole(c.Request.Context(), clinicianID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	eventCtx, _ := c.Get("eventCtx")
	if ctx, ok := eventCtx.(*event.EventContext); ok {
		ctx.OldData = map[string]interface{}{
			"clinician_id": clinicianID,
			"role_id":      roleID,
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListClinicianRoles(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	roles, err := h.service.ListClinicianRoles(c.Request.Context(), clinicianID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(roles))
}
