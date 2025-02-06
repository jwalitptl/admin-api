package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/pkg/model"
	accountService "github.com/jwalitptl/pkg/service/account"

	"github.com/jwalitptl/admin-api/internal/handler"
)

type Handler struct {
	service accountService.Service
}

func NewHandler(service accountService.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/accounts")
	{
		accounts.POST("", h.CreateAccount)
		accounts.GET("", h.ListAccounts)
		accounts.GET("/:id", h.GetAccount)
		accounts.PUT("/:id", h.UpdateAccount)
		accounts.DELETE("/:id", h.DeleteAccount)

		// Organization routes
		accounts.POST("/:id/organizations", h.CreateOrganization)
		accounts.GET("/:id/organizations", h.ListOrganizations)
	}

	organizations := r.Group("/organizations")
	{
		organizations.GET("/:id", h.GetOrganization)
		organizations.PUT("/:id", h.UpdateOrganization)
		organizations.DELETE("/:id", h.DeleteOrganization)
	}
}

type createAccountRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func (h *Handler) CreateAccount(c *gin.Context) {
	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	acc := &model.Account{
		Name:   req.Name,
		Email:  req.Email,
		Status: "active",
	}

	if err := h.service.CreateAccount(c.Request.Context(), acc); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(acc))
}

func (h *Handler) GetAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	acc, err := h.service.GetAccount(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(acc))
}

type updateAccountRequest struct {
	Name   string `json:"name" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	Status string `json:"status" binding:"required"`
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid account ID"))
		return
	}

	var req updateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	acc := &model.Account{
		Base: model.Base{
			ID: id,
		},
		Name:   req.Name,
		Email:  req.Email,
		Status: req.Status,
	}

	if err := h.service.UpdateAccount(c.Request.Context(), acc); err != nil {
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(acc))
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
	accounts, err := h.service.ListAccounts(c.Request.Context())
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
