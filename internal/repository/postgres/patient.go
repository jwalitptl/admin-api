package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type patientRepository struct {
	db *sqlx.DB
}

func NewPatientRepository(db *sqlx.DB) repository.PatientRepository {
	return &patientRepository{db: db}
}

func (r *patientRepository) Create(ctx context.Context, patient *model.Patient) error {
	query := `
		INSERT INTO patients (id, clinic_id, name, email, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	patient.CreatedAt = time.Now()
	patient.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		patient.ID,
		patient.ClinicID,
		patient.Name,
		patient.Email,
		patient.Status,
		patient.CreatedAt,
		patient.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create patient: %w", err)
	}
	return nil
}

func (r *patientRepository) Get(ctx context.Context, id uuid.UUID) (*model.Patient, error) {
	query := `SELECT * FROM patients WHERE id = $1`
	var patient model.Patient
	err := r.db.GetContext(ctx, &patient, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}
	return &patient, nil
}

func (r *patientRepository) Update(ctx context.Context, patient *model.Patient) error {
	query := `UPDATE patients SET name = $1, email = $2, status = $3, updated_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, patient.Name, patient.Email, patient.Status, time.Now(), patient.ID)
	return err
}

func (r *patientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM patients WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *patientRepository) List(ctx context.Context, clinicID uuid.UUID) ([]*model.Patient, error) {
	query := `SELECT * FROM patients WHERE clinic_id = $1`
	var patients []*model.Patient
	err := r.db.SelectContext(ctx, &patients, query, clinicID)
	return patients, err
}

func (r *patientRepository) DeletePatientAppointments(ctx context.Context, patientID uuid.UUID) error {
	query := `DELETE FROM appointments WHERE patient_id = $1`
	_, err := r.db.ExecContext(ctx, query, patientID)
	return err
}
