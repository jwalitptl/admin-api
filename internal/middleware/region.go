package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/service/region"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// ErrRegionRequired is returned when region is required but not provided
var ErrRegionRequired = fmt.Errorf("region is required")

// ErrInvalidRegion is returned when provided region is invalid
var ErrInvalidRegion = fmt.Errorf("invalid region")

// RegionMiddleware handles region-specific logic
type RegionMiddleware struct {
	regionSvc *region.Service
	cache     *cache.Cache
	mu        sync.RWMutex
}

type RegionConfig struct {
	CacheDuration     time.Duration
	CleanupInterval   time.Duration
	DefaultRegionCode string
	IPLookupEnabled   bool
	RequireRegion     bool
}

// RegionConfig represents configuration for region middleware.
// CacheDuration determines how long region configs are cached.
// CleanupInterval determines how often expired cache entries are cleaned up.
// DefaultRegionCode is used when no region is specified and RequireRegion is false.
// IPLookupEnabled enables IP-based region detection.
// RequireRegion makes region specification mandatory.

func DefaultRegionConfig() RegionConfig {
	return RegionConfig{
		CacheDuration:     15 * time.Minute,
		CleanupInterval:   1 * time.Hour,
		DefaultRegionCode: "GLOBAL",
		IPLookupEnabled:   true,
		RequireRegion:     false,
	}
}

func NewRegionMiddleware(regionSvc *region.Service, config RegionConfig) *RegionMiddleware {
	return &RegionMiddleware{
		regionSvc: regionSvc,
		cache:     cache.New(config.CacheDuration, config.CleanupInterval),
	}
}

// DetectRegion middleware detects and validates region
func (m *RegionMiddleware) DetectRegion(cfg RegionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var regionCode string

		// Try to get region from header
		regionCode = c.GetHeader("X-Region")

		// Try to get from query parameter if not in header
		if regionCode == "" {
			regionCode = c.Query("region")
		}

		// Try IP-based detection if enabled
		if regionCode == "" && cfg.IPLookupEnabled {
			regionCode = m.getRegionFromIP(c.ClientIP())
		}

		// Use default if still not found
		if regionCode == "" {
			if cfg.RequireRegion {
				c.Error(ErrRegionRequired)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": ErrRegionRequired.Error(),
				})
				return
			}
			regionCode = cfg.DefaultRegionCode
		}

		// Try to get config from cache
		if cachedConfig, found := m.cache.Get(regionCode); found {
			c.Set("region_config", cachedConfig.(*model.RegionConfig))
			c.Set("region_code", regionCode)
			c.Next()
			return
		}

		// Load region configuration
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		regionConfig, err := m.regionSvc.GetRegionConfig(ctx, regionCode)
		if err != nil {
			if cfg.RequireRegion {
				c.Error(ErrInvalidRegion)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRegion.Error()})
				return
			}
			regionConfig = m.regionSvc.GetDefaultConfig()
		}

		// Cache the config
		m.cache.Set(regionCode, regionConfig, cache.DefaultExpiration)

		// Store in context
		c.Set("region_config", regionConfig)
		c.Set("region_code", regionCode)

		// Apply region-specific settings
		m.applyRegionSettings(c, regionConfig)

		c.Next()
	}
}

func (m *RegionMiddleware) getRegionFromIP(ip string) string {
	// Try cache first
	if cachedRegion, found := m.cache.Get("ip:" + ip); found {
		return cachedRegion.(string)
	}

	// Lookup region
	region, err := m.regionSvc.GetRegionFromIP(context.Background(), ip)
	if err != nil {
		log.Warn().Err(err).Str("ip", ip).Msg("Failed to detect region from IP")
		return ""
	}

	// Cache the result
	m.cache.Set("ip:"+ip, region, cache.DefaultExpiration)

	return region
}

func (m *RegionMiddleware) applyRegionSettings(c *gin.Context, config *model.RegionConfig) {
	// Apply security settings
	if config.SecurityConfig != nil {
		if config.SecurityConfig.MFARequired {
			c.Set("mfa_required", true)
		}

		// Apply rate limits
		if config.APIConfig != nil {
			c.Set("rate_limit", config.APIConfig.RateLimit)
		}
	}

	// Apply audit settings
	if config.AuditConfig != nil {
		c.Set("audit_enabled", true)
		c.Set("audit_fields", config.AuditConfig.RequiredFields)
	}
}
