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

type medicalRecordRepository struct {
	BaseRepository
}

func NewMedicalRecordRepository(base BaseRepository) repository.MedicalRecordRepository {
	return &medicalRecordRepository{base}
}

func (r *medicalRecordRepository) Get(ctx context.Context, id uuid.UUID) (*model.MedicalRecord, error) {
	query := `
		SELECT * FROM medical_records 
		WHERE id = $1 AND deleted_at IS NULL
	`
	var record model.MedicalRecord
	if err := r.GetDB().GetContext(ctx, &record, query, id); err != nil {
		return nil, fmt.Errorf("failed to get medical record: %w", err)
	}

	// Unmarshal JSON fields
	if err := r.unmarshalRecordFields(&record); err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *medicalRecordRepository) List(ctx context.Context, patientID uuid.UUID, filters *model.RecordFilters) ([]*model.MedicalRecord, error) {
	query := `
		SELECT * FROM medical_records 
		WHERE patient_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{patientID}

	if filters.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, filters.Type)
	}

	if !filters.StartDate.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", len(args)+1)
		args = append(args, filters.StartDate)
	}

	if !filters.EndDate.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", len(args)+1)
		args = append(args, filters.EndDate)
	}

	query += " ORDER BY created_at DESC"

	var records []*model.MedicalRecord
	if err := r.GetDB().SelectContext(ctx, &records, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list medical records: %w", err)
	}

	// Unmarshal JSON fields for each record
	for _, record := range records {
		if err := r.unmarshalRecordFields(record); err != nil {
			return nil, err
		}
	}

	return records, nil
}

func (r *medicalRecordRepository) CreateWithAudit(ctx context.Context, record *model.MedicalRecord) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO medical_records (
				id, patient_id, type, description, diagnosis,
				treatment, medications, attachments, access_level,
				created_by, region_code, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`

		medications, err := json.Marshal(record.Medications)
		if err != nil {
			return fmt.Errorf("failed to marshal medications: %w", err)
		}

		attachments, err := json.Marshal(record.Attachments)
		if err != nil {
			return fmt.Errorf("failed to marshal attachments: %w", err)
		}

		_, err = tx.ExecContext(ctx, query,
			record.ID,
			record.PatientID,
			record.Type,
			record.Description,
			record.Diagnosis,
			record.Treatment,
			medications,
			attachments,
			record.AccessLevel,
			record.CreatedBy,
			r.GetRegionFromContext(ctx),
			record.CreatedAt,
			record.UpdatedAt,
		)
		return err
	})
}

func (r *medicalRecordRepository) UpdateWithAudit(ctx context.Context, record *model.MedicalRecord) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		record.UpdatedAt = time.Now()

		query := `
			UPDATE medical_records SET
				type = $1,
				description = $2,
				diagnosis = $3,
				treatment = $4,
				medications = $5,
				attachments = $6,
				access_level = $7,
				updated_at = $8
			WHERE id = $9 AND deleted_at IS NULL
		`

		medications, err := json.Marshal(record.Medications)
		if err != nil {
			return fmt.Errorf("failed to marshal medications: %w", err)
		}

		attachments, err := json.Marshal(record.Attachments)
		if err != nil {
			return fmt.Errorf("failed to marshal attachments: %w", err)
		}

		result, err := tx.ExecContext(ctx, query,
			record.Type,
			record.Description,
			record.Diagnosis,
			record.Treatment,
			medications,
			attachments,
			record.AccessLevel,
			record.UpdatedAt,
			record.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update medical record: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("medical record not found")
		}

		// Create audit log
		auditLog := &model.AuditLog{
			ID:         uuid.New(),
			EntityType: "medical_record",
			EntityID:   record.ID,
			Action:     "update",
			UserID:     record.LastAccessedBy,
			CreatedAt:  time.Now(),
		}

		return r.CreateAuditLog(ctx, tx, auditLog)
	})
}

func (r *medicalRecordRepository) unmarshalRecordFields(record *model.MedicalRecord) error {
	var medications []model.Medication
	if err := json.Unmarshal(record.MedicationsJSON, &medications); err != nil {
		return fmt.Errorf("failed to unmarshal medications: %w", err)
	}
	record.Medications = medications

	var attachments []model.Attachment
	if err := json.Unmarshal(record.AttachmentsJSON, &attachments); err != nil {
		return fmt.Errorf("failed to unmarshal attachments: %w", err)
	}
	record.Attachments = attachments

	return nil
}

func (r *medicalRecordRepository) CreateAuditLog(ctx context.Context, tx *sqlx.Tx, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, entity_type, entity_id, action, user_id,
			ip_address, changes, access_reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := tx.ExecContext(ctx, query,
		log.ID,
		log.EntityType,
		log.EntityID,
		log.Action,
		log.UserID,
		log.IPAddress,
		log.Changes,
		log.AccessReason,
		log.CreatedAt,
	)

	return err
}

func (r *medicalRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE medical_records 
			SET deleted_at = NOW() 
			WHERE id = $1 AND deleted_at IS NULL
		`

		result, err := tx.ExecContext(ctx, query, id)
		if err != nil {
			return fmt.Errorf("failed to delete medical record: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("medical record not found")
		}

		auditLog := &model.AuditLog{
			ID:         uuid.New(),
			EntityType: "medical_record",
			EntityID:   id,
			Action:     "delete",
			CreatedAt:  time.Now(),
		}

		return r.CreateAuditLog(ctx, tx, auditLog)
	})
}
