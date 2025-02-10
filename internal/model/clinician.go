package model

import (
	"time"

	"github.com/google/uuid"
)

type Clinician struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Email      string    `json:"email" db:"email"`
	ClinicID   uuid.UUID `json:"clinic_id" db:"clinic_id"`
	Speciality string    `json:"speciality" db:"speciality"`
	Status     string    `json:"status" db:"status"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
