package model

import (
	"time"

	"github.com/google/uuid"
)

// User status constants
const (
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusPending  = "pending"
	UserStatusLocked   = "locked"
)

// User type constants
const (
	UserTypeAdmin   = "admin"
	UserTypeDoctor  = "doctor"
	UserTypeNurse   = "nurse"
	UserTypeStaff   = "staff"
	UserTypePatient = "patient"
)

// User represents a system user
type User struct {
	Base
	OrganizationID       uuid.UUID  `json:"organization_id" db:"organization_id"`
	Email                string     `json:"email" db:"email"`
	Name                 string     `json:"name" db:"name"`
	Password             string     `json:"password,omitempty" db:"-"`
	PasswordHash         string     `json:"-" db:"password_hash"`
	FirstName            *string    `json:"first_name" db:"first_name"`
	LastName             *string    `json:"last_name" db:"last_name"`
	Phone                *string    `json:"phone" db:"phone"`
	Type                 string     `json:"type" db:"type"`
	Status               string     `json:"status" db:"status"`
	RegionCode           *string    `json:"region_code" db:"region_code"`
	EmailVerified        bool       `json:"email_verified" db:"email_verified"`
	PhoneVerified        bool       `json:"phone_verified" db:"phone_verified"`
	LastLoginAt          *time.Time `json:"last_login_at" db:"last_login_at"`
	LastPasswordChangeAt *time.Time `json:"last_password_change_at" db:"last_password_change_at"`
	FailedLoginAttempts  int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil          *time.Time `json:"locked_until" db:"locked_until"`
	MFAEnabled           bool       `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret            string     `json:"-" db:"mfa_secret"`
	PreferredLanguage    string     `json:"preferred_language" db:"preferred_language"`
	Timezone             string     `json:"timezone" db:"timezone"`
	Settings             JSONMap    `json:"settings" db:"settings"`
	Metadata             JSONMap    `json:"metadata" db:"metadata"`
	LoginAttempts        int        `json:"login_attempts" db:"login_attempts"`
	LastLoginAttempt     time.Time  `json:"last_login_attempt" db:"last_login_attempt"`
}

// UserFilter represents user search parameters
type UserFilter struct {
	BaseFilter
	Type     string    `json:"type" form:"type"`
	ClinicID uuid.UUID `json:"clinic_id" form:"clinic_id"`
}

// CreateUserRequest represents user creation parameters
type CreateUserRequest struct {
	OrganizationID string `json:"organization_id" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Name           string `json:"name" binding:"required"`
	Password       string `json:"password" binding:"required,min=8"`
	Type           string `json:"type" binding:"required"`
}

// UpdateUserRequest represents user update parameters
type UpdateUserRequest struct {
	Name      *string `json:"name"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email" binding:"omitempty,email"`
	Phone     *string `json:"phone"`
	Status    *string `json:"status" binding:"omitempty,oneof=active inactive pending locked"`
	Type      *string `json:"type" binding:"omitempty,oneof=admin doctor nurse staff patient"`
	Settings  JSONMap `json:"settings"`
}

type UserFilters struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	SearchTerm     string    `json:"search_term"`
}

type UserClinic struct {
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	ClinicID  uuid.UUID `db:"clinic_id" json:"clinic_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
