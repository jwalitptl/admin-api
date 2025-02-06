package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jwalitptl/pkg/model"
	"github.com/jwalitptl/pkg/service/auth"

	"github.com/jwalitptl/admin-api/internal/handler"
)

type Handler struct {
	service auth.Service
}

func NewHandler(service auth.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
	}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	tokens, err := h.service.Login(c.Request.Context(), &model.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("invalid email or password"))
			return
		}
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(tokens))
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req refreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	tokens, err := h.service.RefreshToken(c.Request.Context(), &model.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("invalid refresh token"))
		return
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(tokens))
}
