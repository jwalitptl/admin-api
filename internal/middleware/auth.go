package middleware

import (
	"log"
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
		log.Printf("Auth token received: %s", token)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		log.Printf("Validating token: %s", token)

		claims, err := m.authSvc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		log.Printf("Token claims: %+v", claims)

		// Set claims in context
		c.Set("user_id", claims.UserID)
		log.Printf("Setting user_id in context: %#v", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("organization_id", claims.OrganizationID)

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
