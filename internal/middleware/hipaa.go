package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

// HIPAAMiddleware handles HIPAA compliance requirements
type HIPAAMiddleware struct {
	auditSvc *audit.Service
}

// HIPAAConfig represents HIPAA compliance configuration
type HIPAAConfig struct {
	RequiredHeaders []string
	AllowedRoles    []string
	AuditEnabled    bool
	MinTLSVersion   uint16
	RequiredFields  []string
}

// NewHIPAAMiddleware creates a new HIPAA middleware instance
func NewHIPAAMiddleware(auditSvc *audit.Service) *HIPAAMiddleware {
	return &HIPAAMiddleware{
		auditSvc: auditSvc,
	}
}

// Compliance enforces HIPAA compliance requirements
func (m *HIPAAMiddleware) Compliance(config HIPAAConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verify TLS version
		if c.Request.TLS == nil || c.Request.TLS.Version < config.MinTLSVersion {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "HIPAA compliance requires secure transport with minimum TLS version",
			})
			return
		}

		// Verify required headers
		for _, header := range config.RequiredHeaders {
			if c.GetHeader(header) == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "Missing required HIPAA header: " + header,
				})
				return
			}
		}

		// Verify user role
		userType := c.GetString("user_type")
		hasValidRole := false
		for _, role := range config.AllowedRoles {
			if userType == role {
				hasValidRole = true
				break
			}
		}

		if !hasValidRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Insufficient privileges for HIPAA-protected resource",
			})
			return
		}

		// Add HIPAA security headers
		c.Header("X-HIPAA-Audit", "recorded")
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")

		// Log access if audit is enabled
		if config.AuditEnabled {
			m.logAccess(c)
		}

		if c.GetHeader("X-Emergency-Access") == "true" {
			m.logEmergencyAccess(c, c.GetString("user_id"))
		}

		c.Next()
	}
}

func (m *HIPAAMiddleware) logAccess(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.GetString("organization_id")

	m.auditSvc.Log(
		c.Request.Context(),
		uuid.MustParse(userID),
		uuid.MustParse(orgID),
		c.Request.Method,
		c.GetHeader("X-Entity-Type"),
		uuid.MustParse(c.GetHeader("X-Entity-ID")),
		&audit.LogOptions{
			Metadata: map[string]string{
				"ip":        c.ClientIP(),
				"useragent": c.Request.UserAgent(),
				"reason":    c.GetHeader("X-Access-Reason"),
			},
		},
	)
}

func (m *HIPAAMiddleware) logEmergencyAccess(c *gin.Context, userID string) {
	m.auditSvc.Log(
		c.Request.Context(),
		uuid.MustParse(userID),
		uuid.Nil, // No org ID for emergency access
		"emergency_access",
		"emergency",
		uuid.Nil, // No entity ID for emergency access
		&audit.LogOptions{
			Metadata: map[string]string{
				"ip":        c.ClientIP(),
				"useragent": c.Request.UserAgent(),
				"reason":    c.GetHeader("X-Emergency-Reason"),
			},
		},
	)
}
