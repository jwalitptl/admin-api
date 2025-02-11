package user

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/service/user"
	"github.com/jwalitptl/admin-api/pkg/event"
)

type Handler struct {
	service user.UserServicer
	db      *sqlx.DB
}

func NewHandler(service user.UserServicer, db *sqlx.DB) *Handler {
	return &Handler{service: service, db: db}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	fmt.Println("DEBUG: Registering user routes with events")
	users := r.Group("/users")
	{
		users.GET("/me", h.GetMe)
		users.POST("", eventTracker.TrackEvent("user", "create"), h.CreateUser)
		users.PUT("/:id", eventTracker.TrackEvent("user", "update"), h.UpdateUser)
		users.DELETE("/:id", eventTracker.TrackEvent("user", "delete"), h.DeleteUser)
		users.GET("", h.ListUsers)
		users.GET("/:id", h.GetUser)
		users.GET("/:id/clinics", h.ListUserClinics)
		users.GET("/:id/roles", h.ListUserRoles)

		// Role assignments
		users.POST("/:id/roles/:role_id", eventTracker.TrackEvent("user_role", "create"), h.AssignRole)
		users.DELETE("/:id/roles/:role_id", eventTracker.TrackEvent("user_role", "delete"), h.RemoveRole)

		// Clinic assignments
		users.POST("/:id/clinics/:clinic_id", eventTracker.TrackEvent("user_clinic", "create"), h.AssignToClinic)
		users.DELETE("/:id/clinics/:clinic_id", eventTracker.TrackEvent("user_clinic", "delete"), h.RemoveFromClinic)
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {}

func (h *Handler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	user := &model.User{
		OrganizationID: orgID,
		Email:          req.Email,
		Name:           req.Name,
		Password:       req.Password,
		Type:           req.Type,
		Status:         "active",
	}

	if err := h.service.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove sensitive data before sending response
	user.Password = ""
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.FirstName == nil || req.LastName == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "first_name and last_name are required"})
		return
	}

	user := &model.User{
		Base:      model.Base{ID: id},
		Email:     req.Email,
		Type:      req.Type,
		Status:    req.Status,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	if req.OrganizationID != "" {
		orgID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
			return
		}
		user.OrganizationID = orgID
	}

	if err := h.service.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update user: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

func (h *Handler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (h *Handler) ListUsers(c *gin.Context) {
	var filters model.UserFilters

	// Required organization filter
	orgID, err := uuid.Parse(c.Query("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}
	filters.OrganizationID = orgID

	// Optional filters
	if userType := c.Query("type"); userType != "" {
		filters.Type = userType
	}
	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	users, err := h.service.ListUsers(c.Request.Context(), &filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *Handler) AssignRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role ID"})
		return
	}

	if err := h.service.AssignRole(c.Request.Context(), userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role assigned successfully"})
}

func (h *Handler) RemoveRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role ID"})
		return
	}

	if err := h.service.RemoveRole(c.Request.Context(), userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role removed successfully"})
}

func (h *Handler) ListUserRoles(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	roles, err := h.service.ListUserRoles(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func (h *Handler) AssignToClinic(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	clinicID, err := uuid.Parse(c.Param("clinic_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid clinic ID"})
		return
	}

	if err := h.service.AssignToClinic(c.Request.Context(), userID, clinicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "assigned to clinic successfully"})
}

func (h *Handler) RemoveFromClinic(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	clinicID, err := uuid.Parse(c.Param("clinic_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid clinic ID"})
		return
	}

	if err := h.service.RemoveFromClinic(c.Request.Context(), userID, clinicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed from clinic successfully"})
}

func (h *Handler) ListUserClinics(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	clinics, err := h.service.ListUserClinics(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": clinics})
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("GetMe: user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	log.Printf("GetMe: userID from context: %T %v", userID, userID)
	log.Printf("GetMe: raw userID value: %#v", userID)

	uidStr, ok := userID.(string)
	if !ok {
		log.Printf("GetMe: failed type assertion, actual type: %T", userID)
		log.Printf("GetMe: failed value: %#v", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("invalid user ID type: got %T, want string", userID)})
		return
	}

	uid, err := uuid.Parse(uidStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}
