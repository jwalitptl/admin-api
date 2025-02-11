package model

import (
	"time"

	"github.com/google/uuid"
)

type Service struct {
	Base
	ClinicID    uuid.UUID `db:"clinic_id" json:"clinic_id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Duration    int       `db:"duration" json:"duration"` // in minutes
	Price       float64   `db:"price" json:"price"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
