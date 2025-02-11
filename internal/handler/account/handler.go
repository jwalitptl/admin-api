package account

import (
	"fmt"
	"log"
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
		accounts.GET("/:id", h.GetAccount)
		accounts.PUT("/:id", eventTracker.TrackEvent("account", "update"), h.UpdateAccount)
		accounts.DELETE("/:id", eventTracker.TrackEvent("account", "delete"), h.DeleteAccount)
		accounts.GET("", h.ListAccounts)
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
		log.Printf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	account, err := h.service.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		log.Printf("Failed to create account: %v", err)
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(account))
}

func (h *Handler) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	account, err := h.service.GetAccount(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(account))
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	var account model.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}
	account.ID = id

	if err := h.service.UpdateAccount(c.Request.Context(), &account); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) ListAccounts(c *gin.Context) {
	var filters model.AccountFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	accounts, err := h.service.ListAccounts(c.Request.Context(), &filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(accounts))
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
