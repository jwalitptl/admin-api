package model

import (
	"time"

	"github.com/google/uuid"
)

// Base contains common fields for all models
type Base struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Pagination represents common pagination parameters
type Pagination struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// SortOrder represents sorting parameters
type SortOrder struct {
	Field string `json:"field" form:"sort_field"`
	Dir   string `json:"direction" form:"sort_dir"`
}

// BaseFilter contains common filter fields
type BaseFilter struct {
	SearchTerm     string    `json:"search_term" form:"search_term"`
	OrganizationID uuid.UUID `json:"organization_id" form:"organization_id"`
	Status         string    `json:"status" form:"status"`
	StartDate      time.Time `json:"start_date" form:"start_date"`
	EndDate        time.Time `json:"end_date" form:"end_date"`
}

// JSONMap represents a generic JSON object
type JSONMap map[string]interface{}
