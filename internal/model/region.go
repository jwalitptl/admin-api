package model

import (
	"time"

	"github.com/google/uuid"
)

type Region struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Code         string    `json:"code" db:"code"`         // e.g., US, EU, UK, AU
	Name         string    `json:"name" db:"name"`         // e.g., United States, European Union
	Locale       string    `json:"locale" db:"locale"`     // e.g., en-US, fr-FR
	TimeZone     string    `json:"timezone" db:"timezone"` // e.g., America/New_York
	DateFormat   string    `json:"date_format" db:"date_format"`
	CurrencyCode string    `json:"currency_code" db:"currency_code"`

	// Compliance settings
	DataRetentionDays int  `json:"data_retention_days" db:"data_retention_days"`
	GDPR              bool `json:"gdpr_enabled" db:"gdpr_enabled"`
	HIPAA             bool `json:"hipaa_enabled" db:"hipaa_enabled"`
	CCPA              bool `json:"ccpa_enabled" db:"ccpa_enabled"`

	// Feature flags for region-specific features
	Features map[string]bool `json:"features" db:"features"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RegionConfig holds runtime configuration for a region
type RegionConfig struct {
	Region         *Region
	AuditConfig    *AuditConfig
	SecurityConfig *SecurityConfig
	APIConfig      *APIConfig
	Features       map[string]bool
}

type AuditConfig struct {
	RetentionDays   int
	RequiredFields  []string
	SensitiveFields []string
}

type SecurityConfig struct {
	PasswordPolicy  *PasswordPolicy
	MFARequired     bool
	AllowedIPRanges []string
	EncryptionKey   string
}

type APIConfig struct {
	RateLimit       int
	MaxPageSize     int
	RequiredHeaders []string
	AllowedOrigins  []string
}
