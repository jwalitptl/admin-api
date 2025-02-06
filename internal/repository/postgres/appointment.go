package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

// All appointment repository methods here

func (r *appointmentRepository) Create(ctx context.Context, appointment *model.Appointment) error {
	query := `
		INSERT INTO appointments (
			id, clinic_id, clinician_id, patient_id,
			start_time, end_time, status, notes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	appointment.ID = uuid.New()
	appointment.CreatedAt = time.Now()
	appointment.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		appointment.ID,
		appointment.ClinicID,
		appointment.ClinicianID,
		appointment.PatientID,
		appointment.StartTime,
		appointment.EndTime,
		appointment.Status,
		appointment.Notes,
		appointment.CreatedAt,
		appointment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create appointment: %w", err)
	}
	return nil
}

func (r *appointmentRepository) Get(ctx context.Context, id uuid.UUID) (*model.Appointment, error) {
	query := `
		SELECT id, clinic_id, clinician_id, patient_id,
			   start_time, end_time, status, notes,
			   created_at, updated_at
		FROM appointments
		WHERE id = $1
	`
	var appointment model.Appointment
	err := r.db.GetContext(ctx, &appointment, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}
	return &appointment, nil
}

func (r *appointmentRepository) Update(ctx context.Context, appointment *model.Appointment) error {
	query := `
		UPDATE appointments
		SET start_time = $1, end_time = $2, status = $3, notes = $4, updated_at = $5
		WHERE id = $6
	`
	appointment.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		appointment.StartTime,
		appointment.EndTime,
		appointment.Status,
		appointment.Notes,
		appointment.UpdatedAt,
		appointment.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("appointment not found")
	}

	return nil
}

func (r *appointmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM appointments
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("appointment not found")
	}

	return nil
}

func (r *appointmentRepository) List(ctx context.Context, clinicID uuid.UUID, filters map[string]interface{}) ([]*model.Appointment, error) {
	query := `
		SELECT id, clinic_id, clinician_id, patient_id,
			   start_time, end_time, status, notes,
			   created_at, updated_at
		FROM appointments
		WHERE clinic_id = $1
	`
	args := []interface{}{clinicID}
	argCount := 2

	if v, ok := filters["clinician_id"]; ok {
		query += fmt.Sprintf(" AND clinician_id = $%d", argCount)
		args = append(args, v)
		argCount++
	}

	if v, ok := filters["status"]; ok {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, v)
		argCount++
	}

	query += " ORDER BY start_time ASC"

	var appointments []*model.Appointment
	err := r.db.SelectContext(ctx, &appointments, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}
	return appointments, nil
}

func (r *appointmentRepository) CheckConflicts(ctx context.Context, clinicianID uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM appointments
			WHERE clinician_id = $1
			AND status NOT IN ('cancelled', 'completed')
			AND (
				(start_time <= $2 AND end_time > $2)
				OR (start_time < $3 AND end_time >= $3)
				OR (start_time >= $2 AND end_time <= $3)
			)
	`
	args := []interface{}{clinicianID, startTime, endTime}

	if excludeID != nil {
		query += " AND id != $4"
		args = append(args, *excludeID)
	}

	query += ")"

	var hasConflict bool
	err := r.db.GetContext(ctx, &hasConflict, query, args...)
	if err != nil {
		return false, fmt.Errorf("failed to check conflicts: %w", err)
	}
	return hasConflict, nil
}

func (r *appointmentRepository) GetClinicianAppointments(ctx context.Context, clinicianID uuid.UUID, startDate, endDate time.Time) ([]*model.Appointment, error) {
	query := `
		SELECT id, clinic_id, clinician_id, patient_id,
			   start_time, end_time, status, notes,
			   created_at, updated_at
		FROM appointments
		WHERE clinician_id = $1
		AND start_time >= $2
		AND end_time <= $3
		AND status NOT IN ('cancelled', 'completed')
		ORDER BY start_time ASC
	`
	var appointments []*model.Appointment
	err := r.db.SelectContext(ctx, &appointments, query, clinicianID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician appointments: %w", err)
	}
	return appointments, nil
}
