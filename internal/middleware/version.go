package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// APIVersion represents an API version
type APIVersion struct {
	Major       int
	Minor       int
	Path        string
	Deprecation *VersionDeprecation
}

// VersionDeprecation holds deprecation info
type VersionDeprecation struct {
	Date       string
	SunsetDate string
	Info       string
}

// VersionConfig represents version middleware configuration
type VersionConfig struct {
	HeaderName     string
	DefaultVersion string
	Versions       map[string]APIVersion
	Strict         bool
}

func DefaultVersionConfig() VersionConfig {
	return VersionConfig{
		HeaderName:     "Accept-Version",
		DefaultVersion: "1.0",
		Strict:         true,
		Versions: map[string]APIVersion{
			"1.0": {Major: 1, Minor: 0, Path: "/v1"},
			"2.0": {
				Major: 2,
				Minor: 0,
				Path:  "/v2",
				Deprecation: &VersionDeprecation{
					Date:       "2024-01-01",
					SunsetDate: "2024-06-01",
					Info:       "Please upgrade to v3",
				},
			},
		},
	}
}

// Version middleware handles API versioning
func Version(config VersionConfig) gin.HandlerFunc {
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)$`)

	return func(c *gin.Context) {
		requestedVersion := c.GetHeader(config.HeaderName)
		if requestedVersion == "" {
			requestedVersion = config.DefaultVersion
		}

		// Validate version format
		if !versionRegex.MatchString(requestedVersion) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Invalid version format. Use: major.minor",
			})
			return
		}

		version, exists := config.Versions[requestedVersion]
		if !exists && config.Strict {
			c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{
				"error": fmt.Sprintf("API version %s not supported", requestedVersion),
			})
			return
		}

		// Add version info to context
		c.Set("api_version", version)
		c.Set("api_version_string", requestedVersion)

		// Add deprecation headers if needed
		if version.Deprecation != nil {
			c.Header("Deprecation", version.Deprecation.Date)
			if version.Deprecation.SunsetDate != "" {
				c.Header("Sunset", version.Deprecation.SunsetDate)
			}
			if version.Deprecation.Info != "" {
				c.Header("Link", fmt.Sprintf(`<%s>; rel="deprecation"; type="text/html"`,
					version.Deprecation.Info))
			}
		}

		c.Next()
	}
}
