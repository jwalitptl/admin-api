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
	ID                   uuid.UUID         `json:"id" db:"id"`
	ClinicID             uuid.UUID         `json:"clinic_id" db:"clinic_id"`
	OrganizationID       uuid.UUID         `json:"organization_id" db:"organization_id"`
	FirstName            string            `json:"first_name" db:"first_name"`
	LastName             string            `json:"last_name" db:"last_name"`
	Email                string            `json:"email" db:"email"`
	Phone                string            `json:"phone" db:"phone"`
	DateOfBirth          time.Time         `json:"date_of_birth" db:"date_of_birth"`
	Gender               string            `json:"gender" db:"gender"`
	Address              string            `json:"address" db:"address"`
	EmergencyContact     *EmergencyContact `json:"emergency_contact" db:"-"`
	InsuranceInfo        *InsuranceInfo    `json:"insurance_info" db:"-"`
	Status               string            `json:"status" db:"status"`
	CreatedAt            time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at" db:"updated_at"`
	Name                 string            `db:"name" json:"name"`
	PhoneNumber          string            `db:"phone_number" json:"phone_number"`
	EmergencyContactJSON string            `db:"emergency_contact" json:"-"`
	InsuranceInfoJSON    string            `db:"insurance_info" json:"-"`
	UserID               uuid.UUID         `json:"user_id" db:"user_id"`
}

type EmergencyContact struct {
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Phone    string `json:"phone"`
}

type InsuranceInfo struct {
	Provider     string `json:"provider"`
	PolicyNumber string `json:"policy_number"`
	GroupNumber  string `json:"group_number"`
}

type UpdatePatientRequest struct {
	FirstName   *string    `json:"first_name"`
	LastName    *string    `json:"last_name"`
	Email       *string    `json:"email"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Phone       *string    `json:"phone"`
	Address     *string    `json:"address"`
	Status      *string    `json:"status"`
}

type CreatePatientRequest struct {
	ClinicID  string    `json:"clinic_id" validate:"required,uuid"`
	FirstName string    `json:"first_name" validate:"required"`
	LastName  string    `json:"last_name" validate:"required"`
	Email     string    `json:"email" validate:"required,email"`
	Phone     string    `json:"phone" validate:"required"`
	DOB       time.Time `json:"dob" validate:"required"`
	Address   string    `json:"address" validate:"required"`
	Status    string    `json:"status" validate:"required"`
}
