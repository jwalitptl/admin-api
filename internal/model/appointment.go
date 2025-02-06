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
	ClinicID     uuid.UUID         `db:"clinic_id" json:"clinic_id"`
	ClinicianID  uuid.UUID         `db:"clinician_id" json:"clinician_id"`
	PatientID    uuid.UUID         `db:"patient_id" json:"patient_id"`
	StartTime    time.Time         `db:"start_time" json:"start_time"`
	EndTime      time.Time         `db:"end_time" json:"end_time"`
	Status       AppointmentStatus `db:"status" json:"status"`
	Notes        string            `db:"notes" json:"notes,omitempty"`
	CancelReason string            `db:"cancel_reason" json:"cancel_reason,omitempty"`
	CreatedAt    time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at" json:"updated_at"`
}

type CreateAppointmentRequest struct {
	ClinicID    uuid.UUID `json:"clinic_id" binding:"required"`
	ClinicianID uuid.UUID `json:"clinician_id" binding:"required"`
	PatientID   uuid.UUID `json:"patient_id" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	Notes       string    `json:"notes"`
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
