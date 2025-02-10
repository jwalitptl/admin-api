package model

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description" db:"description"`
	OrganizationID *uuid.UUID `json:"organization_id" db:"organization_id"`
	IsSystemRole   bool       `json:"is_system_role" db:"is_system_role"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Permission struct {
	Base
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
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

const (
	// Permission constants
	PermissionCreatePatient     = "create:patient"
	PermissionReadPatient       = "read:patient"
	PermissionUpdatePatient     = "update:patient"
	PermissionDeletePatient     = "delete:patient"
	PermissionCreateAppointment = "create:appointment"
	PermissionReadAppointment   = "read:appointment"
	PermissionUpdateAppointment = "update:appointment"
	PermissionDeleteAppointment = "delete:appointment"
	PermissionManageUsers       = "manage:users"
	PermissionManageRoles       = "manage:roles"
)
