package model

import (
	"time"

	"github.com/google/uuid"
)

type PatientFilters struct {
	ClinicID       uuid.UUID `json:"clinic_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	SearchTerm     string    `json:"search_term"`
	Status         string    `json:"status"`
}

type RecordFilters struct {
	Type      string    `json:"type"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}
