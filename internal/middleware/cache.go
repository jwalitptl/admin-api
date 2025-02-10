package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CacheConfig represents cache control configuration
type CacheConfig struct {
	MaxAge               int
	Private              bool
	NoStore              bool
	MustRevalidate       bool
	ProxyRevalidate      bool
	NoCache              bool
	NoTransform          bool
	StaleWhileRevalidate int
	StaleIfError         int
	Vary                 []string
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxAge:         3600,
		Private:        true,
		MustRevalidate: true,
		Vary:           []string{"Accept", "Authorization"},
	}
}

// Cache adds cache control headers to responses
func Cache(config CacheConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip cache headers for non-GET requests
		if c.Request.Method != "GET" {
			c.Header("Cache-Control", "no-store")
			c.Next()
			return
		}

		directives := make([]string, 0)

		if config.Private {
			directives = append(directives, "private")
		} else {
			directives = append(directives, "public")
		}

		if config.MaxAge > 0 {
			directives = append(directives, "max-age="+string(config.MaxAge))
		}

		if config.NoStore {
			directives = append(directives, "no-store")
		}

		if config.NoCache {
			directives = append(directives, "no-cache")
		}

		if config.MustRevalidate {
			directives = append(directives, "must-revalidate")
		}

		if config.ProxyRevalidate {
			directives = append(directives, "proxy-revalidate")
		}

		if config.NoTransform {
			directives = append(directives, "no-transform")
		}

		if config.StaleWhileRevalidate > 0 {
			directives = append(directives, "stale-while-revalidate="+string(config.StaleWhileRevalidate))
		}

		if config.StaleIfError > 0 {
			directives = append(directives, "stale-if-error="+string(config.StaleIfError))
		}

		if len(directives) > 0 {
			c.Header("Cache-Control", strings.Join(directives, ", "))
		}

		if len(config.Vary) > 0 {
			c.Header("Vary", strings.Join(config.Vary, ", "))
		}

		c.Next()
	}
}
