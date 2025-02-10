package clinic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

type Service struct {
	repo    repository.ClinicRepository
	auditor *audit.Service
}

func NewService(repo repository.ClinicRepository, auditor *audit.Service) *Service {
	return &Service{
		repo:    repo,
		auditor: auditor,
	}
}

func (s *Service) CreateClinic(ctx context.Context, clinic *model.Clinic) error {
	clinic.ID = uuid.New()
	clinic.CreatedAt = time.Now()
	clinic.UpdatedAt = time.Now()
	clinic.Status = "active"

	if err := s.validateClinic(clinic); err != nil {
		return fmt.Errorf("invalid clinic data: %w", err)
	}

	if err := s.repo.Create(ctx, clinic); err != nil {
		return fmt.Errorf("failed to create clinic: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, clinic.OrganizationID, "create", "clinic", clinic.ID, &audit.LogOptions{
		Changes: clinic,
	})

	return nil
}

func (s *Service) GetClinic(ctx context.Context, id uuid.UUID) (*model.Clinic, error) {
	clinic, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinic: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, clinic.OrganizationID, "read", "clinic", id, nil)
	return clinic, nil
}

func (s *Service) UpdateClinic(ctx context.Context, clinic *model.Clinic) error {
	if err := s.validateClinic(clinic); err != nil {
		return fmt.Errorf("invalid clinic data: %w", err)
	}

	clinic.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, clinic); err != nil {
		return fmt.Errorf("failed to update clinic: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, clinic.OrganizationID, "update", "clinic", clinic.ID, &audit.LogOptions{
		Changes: clinic,
	})

	return nil
}

func (s *Service) DeleteClinic(ctx context.Context, id uuid.UUID) error {
	clinic, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get clinic: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete clinic: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, clinic.OrganizationID, "delete", "clinic", id, nil)
	return nil
}

func (s *Service) ListClinics(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error) {
	clinics, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clinics: %w", err)
	}
	return clinics, nil
}

func (s *Service) validateClinic(clinic *model.Clinic) error {
	if clinic.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization ID is required")
	}

	if clinic.Name == "" {
		return fmt.Errorf("clinic name is required")
	}

	if clinic.Location == "" {
		return fmt.Errorf("clinic location is required")
	}

	if clinic.Status != "active" && clinic.Status != "inactive" {
		return fmt.Errorf("invalid clinic status: must be 'active' or 'inactive'")
	}

	return nil
}
