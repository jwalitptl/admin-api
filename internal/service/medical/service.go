package medical

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/pkg/security"
)

const (
	accessLevelPublic  = "public"
	accessLevelPrivate = "private"
	accessLevelHIPAA   = "hipaa"
)

type Service struct {
	repo      repository.MedicalRecordRepository
	encryptor security.Encryptor
	auditor   *audit.Service
}

func NewService(repo repository.MedicalRecordRepository, encryptor security.Encryptor, auditor *audit.Service) *Service {
	return &Service{
		repo:      repo,
		encryptor: encryptor,
		auditor:   auditor,
	}
}

func (s *Service) CreateMedicalRecord(ctx context.Context, record *model.MedicalRecord) error {
	if err := s.validateRecord(record); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}

	record.ID = uuid.New()
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()
	record.LastAccessedAt = time.Now()

	// Encrypt sensitive data
	if err := s.encryptSensitiveData(record); err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	if err := s.repo.CreateWithAudit(ctx, record); err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}

	s.auditor.Log(ctx, record.CreatedBy, uuid.Nil, "create", "medical_record", record.ID, &audit.LogOptions{
		AccessLevel: record.AccessLevel,
	})

	return nil
}

func (s *Service) GetMedicalRecord(ctx context.Context, id uuid.UUID, accessReason string) (*model.MedicalRecord, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	// Decrypt sensitive data
	if err := s.decryptSensitiveData(record); err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Update access metadata
	record.LastAccessedAt = time.Now()
	record.LastAccessedBy = s.getCurrentUserID(ctx)

	s.auditor.Log(ctx, record.LastAccessedBy, uuid.Nil, "read", "medical_record", id, &audit.LogOptions{
		AccessLevel:  record.AccessLevel,
		AccessReason: accessReason,
	})

	return record, nil
}

func (s *Service) UpdateMedicalRecord(ctx context.Context, record *model.MedicalRecord) error {
	if err := s.validateRecord(record); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}

	record.UpdatedAt = time.Now()

	// Encrypt sensitive data
	if err := s.encryptSensitiveData(record); err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	if err := s.repo.UpdateWithAudit(ctx, record); err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), uuid.Nil, "update", "medical_record", record.ID, &audit.LogOptions{
		AccessLevel: record.AccessLevel,
		Changes:     record,
	})

	return nil
}

func (s *Service) ListMedicalRecords(ctx context.Context, patientID uuid.UUID, filters *model.RecordFilters) ([]*model.MedicalRecord, error) {
	records, err := s.repo.List(ctx, patientID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	// Decrypt records
	for _, record := range records {
		if err := s.decryptSensitiveData(record); err != nil {
			return nil, fmt.Errorf("failed to decrypt record %s: %w", record.ID, err)
		}
	}

	return records, nil
}

func (s *Service) validateRecord(record *model.MedicalRecord) error {
	if record.PatientID == uuid.Nil {
		return fmt.Errorf("patient ID is required")
	}

	if record.Type == "" {
		return fmt.Errorf("record type is required")
	}

	if record.AccessLevel == "" {
		return fmt.Errorf("access level is required")
	}

	if !s.isValidAccessLevel(record.AccessLevel) {
		return fmt.Errorf("invalid access level: must be public, private, or hipaa")
	}

	return nil
}

func (s *Service) encryptSensitiveData(record *model.MedicalRecord) error {
	if record.Diagnosis != nil {
		diagnosisJSON, err := json.Marshal(record.Diagnosis)
		if err != nil {
			return err
		}

		encrypted, err := s.encryptor.Encrypt(diagnosisJSON)
		if err != nil {
			return err
		}
		record.Diagnosis = encrypted
	}

	if record.Treatment != nil {
		treatmentJSON, err := json.Marshal(record.Treatment)
		if err != nil {
			return err
		}

		encrypted, err := s.encryptor.Encrypt(treatmentJSON)
		if err != nil {
			return err
		}
		record.Treatment = encrypted
	}

	return nil
}

func (s *Service) decryptSensitiveData(record *model.MedicalRecord) error {
	if record.Diagnosis != nil {
		decrypted, err := s.encryptor.Decrypt(record.Diagnosis)
		if err != nil {
			return err
		}
		record.Diagnosis = decrypted
	}

	if record.Treatment != nil {
		decrypted, err := s.encryptor.Decrypt(record.Treatment)
		if err != nil {
			return err
		}
		record.Treatment = decrypted
	}

	return nil
}

func (s *Service) isValidAccessLevel(level string) bool {
	return level == accessLevelPublic || level == accessLevelPrivate || level == accessLevelHIPAA
}

func (s *Service) getCurrentUserID(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}
