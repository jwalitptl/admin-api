package model

import (
	"time"

	"github.com/google/uuid"
)

type AppointmentStatus string

const (
	AppointmentStatusScheduled AppointmentStatus = "scheduled"
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
	AppointmentStatusCompleted AppointmentStatus = "completed"
)

type Appointment struct {
	Base
	ClinicID        uuid.UUID         `db:"clinic_id" json:"clinic_id"`
	ClinicianID     uuid.UUID         `db:"clinician_id" json:"clinician_id"`
	PatientID       uuid.UUID         `db:"patient_id" json:"patient_id"`
	ServiceID       uuid.UUID         `db:"service_id" json:"service_id"`
	StaffID         uuid.UUID         `db:"staff_id" json:"staff_id"`
	StartTime       time.Time         `db:"start_time" json:"start_time"`
	EndTime         time.Time         `db:"end_time" json:"end_time"`
	Status          AppointmentStatus `db:"status" json:"status"`
	Notes           string            `db:"notes" json:"notes,omitempty"`
	CancelReason    *string           `json:"cancel_reason,omitempty" db:"cancel_reason"`
	CompletedAt     *time.Time        `db:"completed_at" json:"completed_at,omitempty"`
	AppointmentType string            `json:"appointment_type" db:"appointment_type"`
}

type CreateAppointmentRequest struct {
	ClinicID        string    `json:"clinic_id" validate:"required,uuid"`
	PatientID       uuid.UUID `json:"patient_id" validate:"required,uuid"`
	ClinicianID     string    `json:"clinician_id" validate:"required,uuid"`
	ServiceID       uuid.UUID `json:"service_id" validate:"required,uuid"`
	StaffID         uuid.UUID `json:"staff_id" validate:"required,uuid"`
	StartTime       time.Time `json:"start_time" validate:"required,gt=now"`
	EndTime         time.Time `json:"end_time" validate:"required,gtfield=StartTime"`
	AppointmentType string    `json:"appointment_type" validate:"required,oneof=regular followup emergency"`
	Notes           string    `json:"notes" validate:"max=1000"`
}

type UpdateAppointmentRequest struct {
	StartTime    *time.Time         `json:"start_time"`
	EndTime      *time.Time         `json:"end_time"`
	Status       *AppointmentStatus `json:"status"`
	Notes        *string            `json:"notes"`
	CancelReason *string            `json:"cancel_reason"`
}

type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type AppointmentFilters struct {
	ClinicID    uuid.UUID
	ClinicianID uuid.UUID
	PatientID   uuid.UUID
	Status      AppointmentStatus
	StartDate   time.Time
	EndDate     time.Time
}
