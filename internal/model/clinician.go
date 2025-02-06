package model

import (
	"time"

	"github.com/google/uuid"
)

type Clinician struct {
	Base
	Email        string    `db:"email" json:"email"`
	Name         string    `db:"name" json:"name"`
	Password     string    `db:"-" json:"password,omitempty"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Status       string    `db:"status" json:"status"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type ClinicianClinic struct {
	Base
	ClinicianID uuid.UUID  `json:"clinician_id"`
	ClinicID    uuid.UUID  `json:"clinic_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}
