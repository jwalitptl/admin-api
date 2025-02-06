package model

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	Base
	Name           string     `db:"name" json:"name"`
	Description    string     `db:"description" json:"description"`
	OrganizationID *uuid.UUID `db:"organization_id" json:"organization_id,omitempty"`
	IsSystemRole   bool       `db:"is_system_role" json:"is_system_role"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

type Permission struct {
	Base
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type RolePermission struct {
	Base
	RoleID       uuid.UUID `json:"role_id"`
	PermissionID uuid.UUID `json:"permission_id"`
}

type ClinicianRole struct {
	Base
	ClinicianID    uuid.UUID `json:"clinician_id"`
	RoleID         uuid.UUID `json:"role_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
}
