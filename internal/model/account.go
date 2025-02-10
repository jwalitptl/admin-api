package model

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"password_hash" db:"password_hash"`
	Status       string     `json:"status" db:"status"`
	Plan         string     `json:"plan" db:"plan"`
	BillingEmail string     `json:"billing_email" db:"billing_email"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	TrialEndsAt  *time.Time `json:"trial_ends_at" db:"trial_ends_at"`
}

type Organization struct {
	Base
	AccountID string    `db:"account_id" json:"account_id"`
	Name      string    `db:"name" json:"name"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateAccountRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type AccountFilters struct {
	Status    string
	Plan      string
	CreatedAt *time.Time
	Search    string
}

type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusInactive  AccountStatus = "inactive"
	AccountStatusSuspended AccountStatus = "suspended"
)

type OrganizationStatus string

const (
	OrganizationStatusActive   OrganizationStatus = "active"
	OrganizationStatusInactive OrganizationStatus = "inactive"
)
