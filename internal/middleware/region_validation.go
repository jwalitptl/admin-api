package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/model"
)

type RegionValidationMiddleware struct {
	defaultConfig *model.RegionConfig
}

func NewRegionValidationMiddleware(defaultConfig *model.RegionConfig) *RegionValidationMiddleware {
	return &RegionValidationMiddleware{
		defaultConfig: defaultConfig,
	}
}

// ValidateRegionRequirements checks region-specific requirements like GDPR, HIPAA, etc.
func (m *RegionValidationMiddleware) ValidateRegionRequirements() gin.HandlerFunc {
	return func(c *gin.Context) {
		config, exists := c.Get("region_config")
		if !exists {
			c.JSON(http.StatusInternalServerError, handler.NewErrorResponse("region configuration not found"))
			c.Abort()
			return
		}

		regionConfig := config.(*model.RegionConfig)

		// Validate GDPR requirements
		if regionConfig.Region.GDPR {
			if err := m.validateGDPRHeaders(c); err != nil {
				c.JSON(http.StatusForbidden, handler.NewErrorResponse(err.Error()))
				c.Abort()
				return
			}
		}

		// Validate HIPAA requirements
		if regionConfig.Region.HIPAA {
			if err := m.validateHIPAAHeaders(c); err != nil {
				c.JSON(http.StatusForbidden, handler.NewErrorResponse(err.Error()))
				c.Abort()
				return
			}
		}

		// Validate CCPA requirements
		if regionConfig.Region.CCPA {
			if err := m.validateCCPAHeaders(c); err != nil {
				c.JSON(http.StatusForbidden, handler.NewErrorResponse(err.Error()))
				c.Abort()
				return
			}
		}

		// Validate API requirements
		if err := m.validateAPIRequirements(c, regionConfig.APIConfig); err != nil {
			c.JSON(http.StatusForbidden, handler.NewErrorResponse(err.Error()))
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateRegionFeatures checks if requested features are enabled for the region
func (m *RegionValidationMiddleware) ValidateRegionFeatures(requiredFeatures ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, exists := c.Get("region_config")
		if !exists {
			c.JSON(http.StatusInternalServerError, handler.NewErrorResponse("region configuration not found"))
			c.Abort()
			return
		}

		regionConfig := config.(*model.RegionConfig)

		for _, feature := range requiredFeatures {
			if enabled, exists := regionConfig.Region.Features[feature]; !exists || !enabled {
				c.JSON(http.StatusForbidden, handler.NewErrorResponse(fmt.Sprintf("feature %s is not available in your region", feature)))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func (m *RegionValidationMiddleware) validateGDPRHeaders(c *gin.Context) error {
	// Check for required consent
	if consent := c.GetHeader("X-GDPR-Consent"); consent == "" {
		return fmt.Errorf("GDPR consent is required")
	}

	// Check for data processing agreement
	if dpa := c.GetHeader("X-DPA-Version"); dpa == "" {
		return fmt.Errorf("Data Processing Agreement acceptance is required")
	}

	// Check for data transfer mechanism
	if transfer := c.GetHeader("X-Data-Transfer-Mechanism"); transfer == "" {
		return fmt.Errorf("data transfer mechanism must be specified")
	}

	return nil
}

func (m *RegionValidationMiddleware) validateHIPAAHeaders(c *gin.Context) error {
	// Check for BAA (Business Associate Agreement)
	if baa := c.GetHeader("X-BAA-Version"); baa == "" {
		return fmt.Errorf("Business Associate Agreement is required")
	}

	// Check for secure transport
	if c.Request.TLS == nil {
		return fmt.Errorf("HIPAA compliance requires secure transport (HTTPS)")
	}

	// Check for required security headers
	requiredHeaders := []string{
		"Authorization",
		"X-Request-ID",
		"X-Correlation-ID",
	}

	for _, header := range requiredHeaders {
		if value := c.GetHeader(header); value == "" {
			return fmt.Errorf("required HIPAA security header missing: %s", header)
		}
	}

	return nil
}

func (m *RegionValidationMiddleware) validateCCPAHeaders(c *gin.Context) error {
	// Check for privacy notice acceptance
	if notice := c.GetHeader("X-Privacy-Notice-Version"); notice == "" {
		return fmt.Errorf("privacy notice acceptance is required")
	}

	// Check for do-not-sell preference
	if dns := c.GetHeader("X-Do-Not-Sell"); dns != "" {
		// Set context value for downstream handlers
		c.Set("ccpa_do_not_sell", strings.ToLower(dns) == "true")
	}

	return nil
}

func (m *RegionValidationMiddleware) validateAPIRequirements(c *gin.Context, config *model.APIConfig) error {
	// Validate required headers
	for _, header := range config.RequiredHeaders {
		if value := c.GetHeader(header); value == "" {
			return fmt.Errorf("required header missing: %s", header)
		}
	}

	// Validate origin
	origin := c.GetHeader("Origin")
	if origin != "" {
		allowed := false
		for _, allowedOrigin := range config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("origin not allowed: %s", origin)
		}
	}

	return nil
}

func (m *RegionValidationMiddleware) ValidateRegion() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implementation for region validation
		c.Next()
	}
}

func (m *RegionValidationMiddleware) ValidateRequirements() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implementation for requirements validation
		c.Next()
	}
}

func (m *RegionValidationMiddleware) ValidateFeature(featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := c.MustGet("region_config").(*model.RegionConfig)
		if enabled := config.Features[featureName]; !enabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("feature %s not enabled for this region", featureName),
			})
			return
		}
		c.Next()
	}
}
