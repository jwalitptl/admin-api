package region

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

const (
	defaultRegionCode = "GLOBAL"
	cacheExpiry       = 15 * time.Minute
)

type Service struct {
	repo          repository.RegionRepository
	auditor       *audit.Service
	geoIPDB       GeoIPDB
	defaultConfig *model.RegionConfig
	cache         *sync.Map
	cacheTTL      time.Duration
}

type GeoIPDB interface {
	GetCountryCode(ip string) (string, error)
}

type cachedRegion struct {
	config    *model.RegionConfig
	expiresAt time.Time
}

func NewService(repo repository.RegionRepository, geoIPDB GeoIPDB, auditor *audit.Service, defaultConfig *model.RegionConfig) *Service {
	return &Service{
		repo:          repo,
		geoIPDB:       geoIPDB,
		auditor:       auditor,
		defaultConfig: defaultConfig,
		cache:         &sync.Map{},
		cacheTTL:      cacheExpiry,
	}
}

func (s *Service) GetRegionFromIP(ctx context.Context, ipAddr string) (string, error) {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return defaultRegionCode, fmt.Errorf("invalid IP address")
	}

	countryCode, err := s.geoIPDB.GetCountryCode(ipAddr)
	if err != nil {
		return defaultRegionCode, fmt.Errorf("failed to get country code: %w", err)
	}

	regionCode, err := s.repo.GetRegionCodeForCountry(ctx, countryCode)
	if err != nil {
		return defaultRegionCode, fmt.Errorf("failed to get region code: %w", err)
	}

	s.auditor.Log(ctx, nil, nil, "lookup", "region", nil, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"ip":           ipAddr,
			"country_code": countryCode,
			"region_code":  regionCode,
		},
	})

	return regionCode, nil
}

func (s *Service) GetRegionConfig(ctx context.Context, regionCode string) (*model.RegionConfig, error) {
	if regionCode == "" {
		return s.defaultConfig, nil
	}

	// Check cache first
	if cached, ok := s.getFromCache(regionCode); ok {
		return cached, nil
	}

	region, err := s.repo.GetRegion(ctx, regionCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	config := &model.RegionConfig{
		Region: region,
		AuditConfig: &model.AuditConfig{
			RetentionDays:   region.DataRetentionDays,
			RequiredFields:  []string{"user_id", "action", "entity_type"},
			SensitiveFields: []string{"medical_record", "prescription"},
		},
		SecurityConfig: &model.SecurityConfig{
			PasswordPolicy:  s.getPasswordPolicyForRegion(region),
			MFARequired:     region.HIPAA,
			AllowedIPRanges: []string{},
		},
		APIConfig: &model.APIConfig{
			RateLimit:       1000,
			MaxPageSize:     100,
			RequiredHeaders: []string{"X-API-Key"},
			AllowedOrigins:  []string{"*"},
		},
		Features: region.Features,
	}

	// Cache the config
	s.cacheConfig(regionCode, config)

	s.auditor.Log(ctx, nil, nil, "get_config", "region", nil, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"region_code": regionCode,
			"features":    region.Features,
		},
	})

	return config, nil
}

func (s *Service) UpdateRegion(ctx context.Context, region *model.Region) error {
	if err := s.validateRegion(region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	region.UpdatedAt = time.Now()
	if err := s.repo.UpdateRegion(ctx, region); err != nil {
		return fmt.Errorf("failed to update region: %w", err)
	}

	// Invalidate cache
	s.cache.Delete(region.Code)

	s.auditor.Log(ctx, nil, nil, "update", "region", nil, &audit.LogOptions{
		Changes: region,
	})

	return nil
}

func (s *Service) ListRegions(ctx context.Context) ([]*model.Region, error) {
	regions, err := s.repo.ListRegions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	return regions, nil
}

func (s *Service) validateRegion(region *model.Region) error {
	if region.Code == "" {
		return fmt.Errorf("region code is required")
	}

	if region.Name == "" {
		return fmt.Errorf("region name is required")
	}

	if region.Locale == "" {
		return fmt.Errorf("locale is required")
	}

	if region.TimeZone == "" {
		return fmt.Errorf("timezone is required")
	}

	return nil
}

func (s *Service) getPasswordPolicyForRegion(region *model.Region) *model.PasswordPolicy {
	policy := &model.PasswordPolicy{
		MinLength:           8,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
		MaxAge:              90,
		HistoryCount:        5,
		AllowedSpecialChars: "!@#$%^&*()_+-=[]{}|;:,.<>?",
	}

	if region.HIPAA || region.GDPR {
		policy.MinLength = 12
		policy.MaxAge = 60
		policy.HistoryCount = 10
	}

	return policy
}

func (s *Service) getFromCache(regionCode string) (*model.RegionConfig, bool) {
	if val, ok := s.cache.Load(regionCode); ok {
		cached := val.(cachedRegion)
		if time.Now().Before(cached.expiresAt) {
			return cached.config, true
		}
		s.cache.Delete(regionCode)
	}
	return nil, false
}

func (s *Service) cacheConfig(regionCode string, config *model.RegionConfig) {
	s.cache.Store(regionCode, cachedRegion{
		config:    config,
		expiresAt: time.Now().Add(s.cacheTTL),
	})
}
