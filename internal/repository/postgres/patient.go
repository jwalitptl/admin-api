package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type patientRepository struct {
	BaseRepository
}

func NewPatientRepository(base BaseRepository) repository.PatientRepository {
	return &patientRepository{base}
}

func (r *patientRepository) Create(ctx context.Context, patient *model.Patient) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO patients (
				id, clinic_id, organization_id, first_name, last_name,
				email, phone, date_of_birth, gender, address,
				emergency_contact, insurance_info, status, region_code,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`

		patient.ID = uuid.New()
		patient.CreatedAt = time.Now()
		patient.UpdatedAt = time.Now()

		emergencyContact, err := json.Marshal(patient.EmergencyContact)
		if err != nil {
			return fmt.Errorf("failed to marshal emergency contact: %w", err)
		}

		insuranceInfo, err := json.Marshal(patient.InsuranceInfo)
		if err != nil {
			return fmt.Errorf("failed to marshal insurance info: %w", err)
		}

		_, err = tx.ExecContext(ctx, query,
			patient.ID,
			patient.ClinicID,
			patient.OrganizationID,
			patient.FirstName,
			patient.LastName,
			patient.Email,
			patient.Phone,
			patient.DateOfBirth,
			patient.Gender,
			patient.Address,
			emergencyContact,
			insuranceInfo,
			patient.Status,
			r.GetRegionFromContext(ctx),
			patient.CreatedAt,
			patient.UpdatedAt,
		)
		return err
	})
}

func (r *patientRepository) Get(ctx context.Context, id uuid.UUID) (*model.Patient, error) {
	query := `
		SELECT * FROM patients 
		WHERE id = $1 AND deleted_at IS NULL
	`
	var patient model.Patient
	if err := r.db.GetContext(ctx, &patient, query, id); err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}

	// Unmarshal JSON fields
	if err := r.unmarshalPatientFields(&patient); err != nil {
		return nil, err
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

func (r *patientRepository) List(ctx context.Context, filters *model.PatientFilters) ([]*model.Patient, error) {
	query := `
		SELECT * FROM patients 
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if filters.ClinicID != uuid.Nil {
		query += fmt.Sprintf(" AND clinic_id = $%d", len(args)+1)
		args = append(args, filters.ClinicID)
	}

	if filters.OrganizationID != uuid.Nil {
		query += fmt.Sprintf(" AND organization_id = $%d", len(args)+1)
		args = append(args, filters.OrganizationID)
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, filters.Status)
	}

	if filters.SearchTerm != "" {
		query += fmt.Sprintf(` AND (
			first_name ILIKE $%d OR 
			last_name ILIKE $%d OR 
			email ILIKE $%d OR 
			phone ILIKE $%d
		)`, len(args)+1, len(args)+1, len(args)+1, len(args)+1)
		searchTerm := "%" + filters.SearchTerm + "%"
		args = append(args, searchTerm)
	}

	query += " ORDER BY created_at DESC"

	var patients []*model.Patient
	if err := r.db.SelectContext(ctx, &patients, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list patients: %w", err)
	}

	// Unmarshal JSON fields for each patient
	for _, patient := range patients {
		if err := r.unmarshalPatientFields(patient); err != nil {
			return nil, err
		}
	}

	return patients, nil
}

func (r *patientRepository) DeletePatientAppointments(ctx context.Context, patientID uuid.UUID) error {
	query := `DELETE FROM appointments WHERE patient_id = $1`
	_, err := r.db.ExecContext(ctx, query, patientID)
	return err
}

func (r *patientRepository) AddMedicalRecord(ctx context.Context, record *model.MedicalRecord) error {
	query := `
		INSERT INTO medical_records (id, patient_id, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		record.ID,
		record.PatientID,
		record.Description,
		record.CreatedAt,
		record.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add medical record: %w", err)
	}
	return nil
}

func (r *patientRepository) GetMedicalRecords(ctx context.Context, patientID uuid.UUID) ([]*model.MedicalRecord, error) {
	query := `SELECT * FROM medical_records WHERE patient_id = $1`
	var records []*model.MedicalRecord
	err := r.db.SelectContext(ctx, &records, query, patientID)
	return records, err
}

func (r *patientRepository) CreateAppointment(ctx context.Context, appointment *model.Appointment) error {
	query := `
		INSERT INTO appointments (id, patient_id, clinic_id, service_id, clinician_id, start_time, end_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	appointment.CreatedAt = time.Now()
	appointment.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		appointment.ID,
		appointment.PatientID,
		appointment.ClinicID,
		appointment.ServiceID,
		appointment.ClinicianID,
		appointment.StartTime,
		appointment.EndTime,
		appointment.Status,
		appointment.CreatedAt,
		appointment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create appointment: %w", err)
	}
	return nil
}

func (r *patientRepository) UpdateAppointment(ctx context.Context, appointment *model.Appointment) error {
	query := `UPDATE appointments SET start_time = $1, end_time = $2, status = $3, updated_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, appointment.StartTime, appointment.EndTime, appointment.Status, time.Now(), appointment.ID)
	return err
}

func (r *patientRepository) ListAppointments(ctx context.Context, patientID uuid.UUID, filters *model.AppointmentFilters) ([]*model.Appointment, error) {
	query := `SELECT * FROM appointments WHERE patient_id = $1`
	var appointments []*model.Appointment
	err := r.db.SelectContext(ctx, &appointments, query, patientID)
	return appointments, err
}

func (r *patientRepository) unmarshalPatientFields(patient *model.Patient) error {
	var emergencyContact model.EmergencyContact
	if err := json.Unmarshal([]byte(patient.EmergencyContactJSON), &emergencyContact); err != nil {
		return fmt.Errorf("failed to unmarshal emergency contact: %w", err)
	}
	patient.EmergencyContact = &emergencyContact

	var insuranceInfo model.InsuranceInfo
	if err := json.Unmarshal([]byte(patient.InsuranceInfoJSON), &insuranceInfo); err != nil {
		return fmt.Errorf("failed to unmarshal insurance info: %w", err)
	}
	patient.InsuranceInfo = &insuranceInfo

	return nil
}
