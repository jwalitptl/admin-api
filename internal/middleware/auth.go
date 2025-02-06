package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jwalitptl/pkg/service/auth"
	"github.com/jwalitptl/pkg/service/rbac"

	"github.com/jwalitptl/admin-api/internal/handler"
)

type AuthMiddleware struct {
	rbacService rbac.Service
	authService auth.Service
}

func NewAuthMiddleware(rbacService rbac.Service, authService auth.Service) *AuthMiddleware {
	return &AuthMiddleware{
		rbacService: rbacService,
		authService: authService,
	}
}

// Authenticate verifies the JWT token and sets clinician info in context
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("missing authorization header"))
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("invalid authorization format"))
			c.Abort()
			return
		}

		claims, err := m.authService.ValidateToken(c.Request.Context(), parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("invalid token"))
			c.Abort()
			return
		}

		// Set clinician info in context
		c.Set("clinicianID", claims.ClinicianID)
		c.Set("clinicianEmail", claims.Email)
		c.Next()
	}
}

// RequirePermission checks if the clinician has the required permission
func (m *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clinicianID, err := uuid.Parse(c.GetString("clinicianID"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, handler.NewErrorResponse("invalid clinician ID"))
			c.Abort()
			return
		}

		orgID := c.GetHeader("X-Organization-ID")
		if orgID == "" {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("organization ID is required"))
			c.Abort()
			return
		}

		organizationID, err := uuid.Parse(orgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid organization ID"))
			c.Abort()
			return
		}

		hasPermission, err := m.rbacService.HasPermission(c.Request.Context(), clinicianID, permission, organizationID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, handler.NewErrorResponse("failed to check permission"))
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, handler.NewErrorResponse("permission denied"))
			c.Abort()
			return
		}

		c.Next()
	}
}
