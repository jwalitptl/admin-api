package rbac

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	rbacService "github.com/jwalitptl/admin-api/internal/service/rbac"
	"github.com/jwalitptl/admin-api/pkg/event"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
)

type Handler struct {
	service    rbacService.Service
	outboxRepo postgres.OutboxRepository
}

func NewHandler(service rbacService.Service, outboxRepo postgres.OutboxRepository) *Handler {
	return &Handler{
		service:    service,
		outboxRepo: outboxRepo,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	rbac := r.Group("/rbac")
	{
		roles := rbac.Group("/roles")
		{
			roles.POST("", h.CreateRole)
			roles.GET("", h.ListRoles)
			roles.GET("/:id", h.GetRole)
			roles.PUT("/:id", h.UpdateRole)
			roles.DELETE("/:id", h.DeleteRole)
			roles.POST("/:id/permissions", h.AssignPermissionToRole)
			roles.DELETE("/:id/permissions/:permission", h.RemovePermissionFromRole)
		}

		users := rbac.Group("/users")
		{
			users.POST("/:id/roles", h.AssignRoleToClinician)
			users.DELETE("/:id/roles/:roleId", h.RemoveRoleFromClinician)
		}

		// Permissions
		rbac.GET("/permissions", h.ListPermissions)
		rbac.POST("/permissions", h.CreatePermission)
		rbac.GET("/permissions/:id", h.GetPermission)
		rbac.PUT("/permissions/:id", h.UpdatePermission)
		rbac.DELETE("/permissions/:id", h.DeletePermission)
	}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	rbac := r.Group("/rbac")
	{
		// Roles
		rbac.POST("/roles", eventTracker.TrackEvent("rbac", "role_create"), h.CreateRole)
		rbac.PUT("/roles/:id", eventTracker.TrackEvent("rbac", "role_update"), h.UpdateRole)
		rbac.DELETE("/roles/:id", eventTracker.TrackEvent("rbac", "role_delete"), h.DeleteRole)

		// Permissions
		rbac.POST("/roles/:id/permissions", eventTracker.TrackEvent("rbac", "permission_assign"), h.AssignPermissionToRole)
		rbac.DELETE("/roles/:id/permissions/:permission_id", eventTracker.TrackEvent("rbac", "permission_remove"), h.RemovePermissionFromRole)

		// Non-tracked endpoints
		rbac.GET("/roles", h.ListRoles)
		rbac.GET("/roles/:id", h.GetRole)
		rbac.GET("/roles/:id/permissions", h.ListRolePermissions)
	}
}

type createRoleRequest struct {
	Name           string  `json:"name" binding:"required"`
	Description    string  `json:"description"`
	OrganizationID *string `json:"organization_id"`
	IsSystemRole   bool    `json:"is_system_role"`
}

func (h *Handler) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	var orgID *uuid.UUID
	if req.OrganizationID != nil {
		id, err := uuid.Parse(*req.OrganizationID)
		if err != nil {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
			return
		}
		orgID = &id
	}

	role := &model.Role{
		ID:             uuid.New(),
		Name:           req.Name,
		Description:    req.Description,
		OrganizationID: orgID,
		IsSystemRole:   req.IsSystemRole,
	}

	if err := h.service.CreateRole(c.Request.Context(), role); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Create outbox event
	payload, err := json.Marshal(role)
	if err != nil {
		log.Printf("Failed to marshal role for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(c.Request.Context(), &model.OutboxEvent{
			EventType: "ROLE_CREATE",
			Payload:   payload,
		}); err != nil {
			log.Printf("Failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(role))
}

func (h *Handler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	role, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(role))
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	var role model.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	role.ID = id
	if err := h.service.UpdateRole(c.Request.Context(), &role); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(role))
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	if err := h.service.DeleteRole(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(roles))
}

type createPermissionRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) CreatePermission(c *gin.Context) {
	var req createPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	permission := &model.Permission{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.service.CreatePermission(c.Request.Context(), permission); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(permission))
}

func (h *Handler) GetPermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid permission ID"))
		return
	}

	permission, err := h.service.GetPermission(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(permission))
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid permission ID"))
		return
	}

	var req createPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	permission := &model.Permission{
		Base: model.Base{
			ID: id,
		},
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.service.UpdatePermission(c.Request.Context(), permission); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(permission))
}

func (h *Handler) DeletePermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid permission ID"))
		return
	}

	if err := h.service.DeletePermission(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(permissions))
}

func (h *Handler) AssignPermissionToRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	permissionID, err := uuid.Parse(c.Param("permission"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid permission ID"))
		return
	}

	if err := h.service.AssignPermissionToRole(c.Request.Context(), roleID, permissionID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) RemovePermissionFromRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	permissionID, err := uuid.Parse(c.Param("permission"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid permission ID"))
		return
	}

	if err := h.service.RemovePermissionFromRole(c.Request.Context(), roleID, permissionID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListRolePermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid role ID"))
		return
	}

	permissions, err := h.service.ListRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(permissions))
}

func (h *Handler) AssignRoleToClinician(c *gin.Context) {
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

	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	if err := h.service.AssignRoleToClinician(c.Request.Context(), clinicianID, roleID, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) RemoveRoleFromClinician(c *gin.Context) {
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

	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	if err := h.service.RemoveRoleFromClinician(c.Request.Context(), clinicianID, roleID, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListClinicianRoles(c *gin.Context) {
	clinicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid clinician ID"))
		return
	}

	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	roles, err := h.service.ListClinicianRoles(c.Request.Context(), clinicianID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(roles))
}
