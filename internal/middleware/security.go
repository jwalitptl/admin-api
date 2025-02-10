package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig represents security headers configuration
type SecurityConfig struct {
	HSTS                  bool
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	FrameOptions          string
	ContentTypeOptions    string
	XSSProtection         string
	ReferrerPolicy        string
	CSPDirectives         []string
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		HSTS:                  true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		FrameOptions:          "DENY",
		ContentTypeOptions:    "nosniff",
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		CSPDirectives: []string{
			"default-src 'self'",
			"img-src 'self' data: https:",
			"script-src 'self'",
			"style-src 'self' 'unsafe-inline'",
			"connect-src 'self'",
			"frame-ancestors 'none'",
		},
	}
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders(config SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS
		if config.HSTS {
			value := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
			if config.HSTSIncludeSubdomains {
				value += "; includeSubDomains"
			}
			c.Header("Strict-Transport-Security", value)
		}

		// Other security headers
		c.Header("X-Frame-Options", config.FrameOptions)
		c.Header("X-Content-Type-Options", config.ContentTypeOptions)
		c.Header("X-XSS-Protection", config.XSSProtection)
		c.Header("Referrer-Policy", config.ReferrerPolicy)

		// Content Security Policy
		if len(config.CSPDirectives) > 0 {
			c.Header("Content-Security-Policy", strings.Join(config.CSPDirectives, "; "))
		}

		c.Next()
	}
}
