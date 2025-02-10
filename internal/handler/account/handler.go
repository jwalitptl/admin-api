package account

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	accountService "github.com/jwalitptl/admin-api/internal/service/account"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/pkg/event"
)

type Handler struct {
	service accountService.AccountServicer
}

func NewHandler(service accountService.AccountServicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	accounts := r.Group("/accounts")
	{
		accounts.POST("", eventTracker.TrackEvent("account", "create"), h.CreateAccount)
		accounts.PUT("/:id", eventTracker.TrackEvent("account", "update"), h.UpdateAccount)
		accounts.DELETE("/:id", eventTracker.TrackEvent("account", "delete"), h.DeleteAccount)
		accounts.GET("", h.ListAccounts)
		accounts.GET("/:id", h.GetAccount)

		// Organization routes
		accounts.POST("/:id/organizations", eventTracker.TrackEvent("organization", "create"), h.CreateOrganization)
		accounts.GET("/:id/organizations", h.ListOrganizations)
	}

	organizations := r.Group("/organizations")
	{
		organizations.GET("/:id", h.GetOrganization)
		organizations.PUT("/:id", h.UpdateOrganization)
		organizations.DELETE("/:id", h.DeleteOrganization)
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/accounts")
	{
		accounts.POST("", h.CreateAccount)               // Create new account/shop
		accounts.GET("", h.ListAccounts)                 // List all accounts (admin only)
		accounts.GET("/:id", h.GetAccount)               // Get account details
		accounts.PUT("/:id", h.UpdateAccount)            // Update account settings
		accounts.DELETE("/:id", h.DeleteAccount)         // Delete/Deactivate account
		accounts.POST("/:id/subscription", h.UpdatePlan) // Update subscription plan
	}
}

func (h *Handler) CreateAccount(c *gin.Context) {
	var req model.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	user, err := h.service.GetAccount(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	var user model.Account
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.ID = id
	if err := h.service.UpdateAccount(c.Request.Context(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	var filters model.AccountFilters

	// Parse organization ID if provided
	if orgID := c.Query("organization_id"); orgID != "" {
		filters.Search = orgID // or handle differently based on AccountFilters structure
	}

	filters.Status = c.Query("status")
	filters.Plan = c.Query("plan")
	filters.Search = c.Query("search")

	users, err := h.service.ListAccounts(c.Request.Context(), &filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

type createOrganizationRequest struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status" binding:"required"`
}

func (h *Handler) CreateOrganization(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	var req createOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	org := &model.Organization{
		AccountID: accountID.String(),
		Name:      req.Name,
		Status:    req.Status,
	}

	if err := h.service.CreateOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Set event context
	fmt.Printf("DEBUG: Setting event context for organization creation\n")
	eventCtx, exists := c.Get("eventCtx")
	if exists {
		if ctx, ok := eventCtx.(*event.EventContext); ok {
			ctx.NewData = org
			fmt.Printf("DEBUG: Event context set with organization data\n")
		}
	} else {
		fmt.Printf("DEBUG: No event context found\n")
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(org))
}

func (h *Handler) GetOrganization(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	org, err := h.service.GetOrganization(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(org))
}

type updateOrganizationRequest struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status" binding:"required"`
}

func (h *Handler) UpdateOrganization(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	var req updateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	org := &model.Organization{
		Base: model.Base{
			ID: id,
		},
		Name:   req.Name,
		Status: req.Status,
	}

	if err := h.service.UpdateOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(org))
}

func (h *Handler) DeleteOrganization(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
		return
	}

	if err := h.service.DeleteOrganization(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListOrganizations(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	orgs, err := h.service.ListOrganizations(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(orgs))
}

func (h *Handler) UpdatePlan(c *gin.Context) {
	// Implementation of UpdatePlan method
}
