package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MedicalRecord struct {
	Base
	PatientID       uuid.UUID       `db:"patient_id" json:"patient_id"`
	Type            string          `db:"type" json:"type"`
	Description     string          `db:"description" json:"description"`
	Diagnosis       json.RawMessage `db:"diagnosis" json:"diagnosis"`
	Treatment       json.RawMessage `db:"treatment" json:"treatment"`
	MedicationsJSON json.RawMessage `db:"medications" json:"medications"`
	AttachmentsJSON json.RawMessage `db:"attachments" json:"attachments"`
	Medications     []Medication    `json:"-"`
	Attachments     []Attachment    `json:"-"`
	AccessLevel     string          `db:"access_level" json:"access_level"`
	CreatedBy       uuid.UUID       `db:"created_by" json:"created_by"`
	LastAccessedBy  uuid.UUID       `db:"last_accessed_by" json:"last_accessed_by"`
	LastAccessedAt  time.Time       `db:"last_accessed_at" json:"last_accessed_at"`
}

type Medication struct {
	Name     string `json:"name"`
	Dosage   string `json:"dosage"`
	Schedule string `json:"schedule"`
}

type Attachment struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Type       string    `json:"type" db:"type"`
	Path       string    `json:"path" db:"path"`
	UploadedBy uuid.UUID `json:"uploaded_by" db:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
}
