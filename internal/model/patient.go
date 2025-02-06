package model

import (
	"time"

	"github.com/google/uuid"
)

type PatientStatus string

const (
	PatientStatusActive   PatientStatus = "active"
	PatientStatusInactive PatientStatus = "inactive"
)

type Patient struct {
	Base
	ClinicID  uuid.UUID `db:"clinic_id" json:"clinic_id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreatePatientRequest struct {
	ClinicID string `json:"clinic_id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Status   string `json:"status" binding:"required"`
}

type UpdatePatientRequest struct {
	FirstName   *string    `json:"first_name"`
	LastName    *string    `json:"last_name"`
	Email       *string    `json:"email" binding:"omitempty,email"`
	Phone       *string    `json:"phone"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Gender      *string    `json:"gender"`
	Address     *string    `json:"address"`
	Status      *string    `json:"status"`
	Notes       *string    `json:"notes"`
}
