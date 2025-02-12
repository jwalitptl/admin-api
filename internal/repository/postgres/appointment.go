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

type appointmentRepository struct {
	BaseRepository
}

func NewAppointmentRepository(base BaseRepository) repository.AppointmentRepository {
	return &appointmentRepository{base}
}

// All appointment repository methods here

func (r *appointmentRepository) Create(ctx context.Context, appointment *model.Appointment) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO appointments (
				id, clinic_id, patient_id, clinician_id, staff_id, service_id,
				appointment_type, status, start_time, end_time, notes,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)
		`

		_, err := tx.ExecContext(ctx, query,
			appointment.ID,
			appointment.ClinicID,
			appointment.PatientID,
			appointment.ClinicianID,
			appointment.StaffID,
			appointment.ServiceID,
			appointment.AppointmentType,
			appointment.Status,
			appointment.StartTime,
			appointment.EndTime,
			appointment.Notes,
			time.Now(),
			time.Now(),
		)
		return err
	})
}

func (r *appointmentRepository) Get(ctx context.Context, id uuid.UUID) (*model.Appointment, error) {
	query := `
		SELECT id, clinic_id, patient_id, clinician_id, staff_id, service_id,
			   appointment_type, status, start_time, end_time, notes,
			   created_at, updated_at, deleted_at
		FROM appointments 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var appointment model.Appointment
	if err := r.GetDB().GetContext(ctx, &appointment, query, id); err != nil {
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}

	return &appointment, nil
}

func (r *appointmentRepository) Update(ctx context.Context, appointment *model.Appointment) error {
	query := `
		UPDATE appointments SET
			start_time = $1,
			end_time = $2,
			status = $3,
			notes = $4,
			appointment_type = $5,
			updated_at = $6
		WHERE id = $7 AND deleted_at IS NULL
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		appointment.StartTime,
		appointment.EndTime,
		appointment.Status,
		appointment.Notes,
		appointment.AppointmentType,
		time.Now(),
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
		UPDATE appointments 
		SET deleted_at = NOW() 
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.GetDB().ExecContext(ctx, query, id)
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

func (r *appointmentRepository) List(ctx context.Context, filters *model.AppointmentFilters) ([]*model.Appointment, error) {
	query := `
		SELECT * FROM appointments 
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if !filters.StartDate.IsZero() {
		query += fmt.Sprintf(" AND start_time >= $%d", len(args)+1)
		args = append(args, filters.StartDate)
	}

	if !filters.EndDate.IsZero() {
		query += fmt.Sprintf(" AND end_time <= $%d", len(args)+1)
		args = append(args, filters.EndDate)
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, filters.Status)
	}

	if filters.ClinicianID != uuid.Nil {
		query += fmt.Sprintf(" AND clinician_id = $%d", len(args)+1)
		args = append(args, filters.ClinicianID)
	}

	query += " ORDER BY start_time ASC"

	var appointments []*model.Appointment
	if err := r.GetDB().SelectContext(ctx, &appointments, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}

	return appointments, nil
}

func (r *appointmentRepository) FindConflictingAppointments(ctx context.Context, staffID uuid.UUID, start, end time.Time) ([]*model.Appointment, error) {
	query := `
		SELECT * FROM appointments 
		WHERE clinician_id = $1 
		AND deleted_at IS NULL
		AND status != 'cancelled'
		AND region_code = $2
		AND (
			(start_time <= $3 AND end_time > $3) OR
			(start_time < $4 AND end_time >= $4) OR
			(start_time >= $3 AND end_time <= $4)
		)
	`

	var appointments []*model.Appointment
	if err := r.GetDB().SelectContext(ctx, &appointments, query,
		staffID,
		r.GetRegionFromContext(ctx),
		start,
		end,
	); err != nil {
		return nil, fmt.Errorf("failed to find conflicting appointments: %w", err)
	}

	return appointments, nil
}

func (r *appointmentRepository) CheckConflicts(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM appointments
		WHERE clinician_id = $1
		AND status NOT IN ('cancelled', 'completed')
		AND (
			(start_time <= $2 AND end_time > $2)
			OR (start_time < $3 AND end_time >= $3)
			OR (start_time >= $2 AND end_time <= $3)
		)
	`
	args := []interface{}{userID, startTime, endTime}

	if excludeID != nil {
		query += " AND id != $4"
		args = append(args, *excludeID)
	}

	var hasConflict bool
	err := r.GetDB().GetContext(ctx, &hasConflict, query, args...)
	if err != nil {
		return false, fmt.Errorf("failed to check conflicts: %w", err)
	}
	return hasConflict, nil
}

func (r *appointmentRepository) GetClinicianAppointments(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*model.Appointment, error) {
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
	err := r.GetDB().SelectContext(ctx, &appointments, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician appointments: %w", err)
	}
	return appointments, nil
}

func (r *appointmentRepository) GetClinicianSchedule(ctx context.Context, clinicianID uuid.UUID, date time.Time) ([]*model.TimeSlot, error) {
	query := `
		SELECT start_time, end_time 
		FROM clinician_schedules 
		WHERE clinician_id = $1 
		AND date(start_time) = date($2)
		ORDER BY start_time
	`
	var slots []*model.TimeSlot
	err := r.GetDB().SelectContext(ctx, &slots, query, clinicianID, date)
	return slots, err
}
