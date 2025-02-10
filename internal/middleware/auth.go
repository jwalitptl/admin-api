package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jwalitptl/admin-api/internal/service/auth"
	"github.com/jwalitptl/admin-api/internal/service/rbac"
)

type AuthMiddleware struct {
	rbacService rbac.Service
	authSvc     *auth.Service
}

func NewAuthMiddleware(rbacService rbac.Service, authSvc *auth.Service) *AuthMiddleware {
	return &AuthMiddleware{
		rbacService: rbacService,
		authSvc:     authSvc,
	}
}

// Authenticate verifies the JWT token and sets clinician info in context
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		// Remove Bearer prefix
		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := m.authSvc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		// Add claims to context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("user_type", claims.Type)
		c.Set("organization_id", claims.OrganizationID)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}

// RequirePermission checks if the clinician has the required permission
func (m *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "user not authenticated",
			})
			return
		}

		// Get permissions from context
		permissions, exists := c.Get("permissions")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "no permissions found",
			})
			return
		}

		// Check if user has required permission
		hasPermission := false
		for _, p := range permissions.([]string) {
			if p == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "permission denied",
			})
			return
		}

		c.Next()
	}
}

// RequireRole middleware checks if the authenticated user has the required role
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		hasRole := false
		for _, role := range roles {
			if strings.EqualFold(userType.(string), role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) ValidatePermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get required permissions from route metadata
		requiredPerms := c.GetStringSlice("required_permissions")
		if len(requiredPerms) == 0 {
			c.Next()
			return
		}

		// Get user permissions from context
		userPerms := c.GetStringSlice("permissions")

		// Check if user has required permissions
		for _, required := range requiredPerms {
			hasPermission := false
			for _, userPerm := range userPerms {
				if required == userPerm {
					hasPermission = true
					break
				}
			}
			if !hasPermission {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}
		}

		c.Next()
	}
}
