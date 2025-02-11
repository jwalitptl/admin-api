package model

import (
	"time"

	"github.com/google/uuid"
)

type Clinic struct {
	Base
	OrganizationID uuid.UUID `db:"organization_id" json:"organization_id"`
	Name           string    `db:"name" json:"name"`
	Location       string    `db:"location" json:"location"`
	Status         string    `db:"status" json:"status"`
	RegionCode     string    `db:"region_code" json:"region_code"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type ClinicStaff struct {
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	ClinicID  uuid.UUID `db:"clinic_id" json:"clinic_id"`
	Role      string    `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type AssignStaffRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}
