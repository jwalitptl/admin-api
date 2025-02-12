package clinic

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

type ClinicServicer interface {
	CreateClinic(ctx context.Context, clinic *model.Clinic) error
	GetClinic(ctx context.Context, id uuid.UUID) (*model.Clinic, error)
	UpdateClinic(ctx context.Context, clinic *model.Clinic) error
	DeleteClinic(ctx context.Context, id uuid.UUID) error
	ListClinics(ctx context.Context, organizationID uuid.UUID, search, status string) ([]*model.Clinic, error)
	AssignStaff(ctx context.Context, clinicID, userID uuid.UUID, role string) error
	ListStaff(ctx context.Context, clinicID uuid.UUID) ([]*model.ClinicStaff, error)
	RemoveStaff(ctx context.Context, clinicID, userID uuid.UUID) error
	CreateService(ctx context.Context, service *model.Service) error
	ListServices(ctx context.Context, clinicID uuid.UUID, search, isActive string) ([]*model.Service, error)
	UpdateService(ctx context.Context, service *model.Service) error
	DeleteService(ctx context.Context, clinicID, serviceID uuid.UUID) error
	DeactivateService(ctx context.Context, clinicID, serviceID uuid.UUID) error
	GetService(ctx context.Context, clinicID, serviceID uuid.UUID) (*model.Service, error)
	DeleteClinicStaff(ctx context.Context, clinicID uuid.UUID) error
}

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

func (s *Service) ListClinics(ctx context.Context, organizationID uuid.UUID, search, status string) ([]*model.Clinic, error) {
	ctx = context.WithValue(ctx, "search", search)
	ctx = context.WithValue(ctx, "status", status)
	clinics, err := s.repo.List(ctx, organizationID)
	if err != nil {
		log.Printf("Debug - Repository error: %v", err)
		return nil, fmt.Errorf("failed to list clinics: %w", err)
	}
	log.Printf("Debug - Found %d clinics", len(clinics))
	return clinics, nil
}

func (s *Service) validateClinic(clinic *model.Clinic) error {
	if clinic.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization ID is required")
	}

	if clinic.Name == "" {
		return fmt.Errorf("clinic name is required")
	}

	// Check for duplicate name in the same organization
	existingClinics, err := s.repo.List(context.Background(), clinic.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate names: %w", err)
	}

	for _, existing := range existingClinics {
		if existing.Name == clinic.Name && existing.ID != clinic.ID {
			return fmt.Errorf("clinic with name '%s' already exists in this organization", clinic.Name)
		}
	}

	if clinic.Location == "" {
		return fmt.Errorf("clinic location is required")
	}

	if clinic.Status != "active" && clinic.Status != "inactive" {
		return fmt.Errorf("invalid clinic status: must be 'active' or 'inactive'")
	}

	return nil
}

func (s *Service) AssignStaff(ctx context.Context, clinicID, userID uuid.UUID, role string) error {
	staff := &model.ClinicStaff{
		ClinicID:  clinicID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
	}
	return s.repo.AssignStaff(ctx, staff)
}

func (s *Service) ListStaff(ctx context.Context, clinicID uuid.UUID) ([]*model.ClinicStaff, error) {
	return s.repo.ListStaff(ctx, clinicID)
}

func (s *Service) RemoveStaff(ctx context.Context, clinicID, userID uuid.UUID) error {
	return s.repo.RemoveStaff(ctx, clinicID, userID)
}

func (s *Service) CreateService(ctx context.Context, service *model.Service) error {
	service.ID = uuid.New()
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()
	service.Status = "active"

	if err := s.validateService(service); err != nil {
		return fmt.Errorf("invalid service data: %w", err)
	}

	if err := s.repo.CreateService(ctx, service); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, service.ClinicID, "create", "service", service.ID, &audit.LogOptions{
		Changes: service,
	})

	return nil
}

func (s *Service) ListServices(ctx context.Context, clinicID uuid.UUID, search, isActive string) ([]*model.Service, error) {
	ctx = context.WithValue(ctx, "search", search)
	ctx = context.WithValue(ctx, "is_active", isActive)
	services, err := s.repo.ListServices(ctx, clinicID)
	if err != nil {
		log.Printf("Debug - Repository error: %v", err)
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	log.Printf("Debug - Found %d services", len(services))
	return services, nil
}

func (s *Service) UpdateService(ctx context.Context, service *model.Service) error {
	if err := s.validateService(service); err != nil {
		return fmt.Errorf("invalid service data: %w", err)
	}

	service.UpdatedAt = time.Now()
	if err := s.repo.UpdateService(ctx, service); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, service.ClinicID, "update", "service", service.ID, &audit.LogOptions{
		Changes: service,
	})

	return nil
}

func (s *Service) DeleteService(ctx context.Context, clinicID, serviceID uuid.UUID) error {
	// Skip getting the service since we don't need it
	if err := s.repo.DeleteService(ctx, serviceID); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}
	return nil
}

func (s *Service) DeactivateService(ctx context.Context, clinicID, serviceID uuid.UUID) error {
	service, err := s.repo.GetService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	service.Status = "inactive"
	if err := s.repo.UpdateService(ctx, service); err != nil {
		return fmt.Errorf("failed to deactivate service: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, clinicID, "deactivate", "service", serviceID, &audit.LogOptions{
		Changes: service,
	})

	return nil
}

func (s *Service) validateService(service *model.Service) error {
	if service.ClinicID == uuid.Nil {
		return fmt.Errorf("clinic ID is required")
	}

	if service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Check for duplicate name in the same clinic
	existingServices, err := s.repo.ListServices(context.Background(), service.ClinicID)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate names: %w", err)
	}

	for _, existing := range existingServices {
		if existing.Name == service.Name && existing.ID != service.ID {
			return fmt.Errorf("service with name '%s' already exists in this clinic", service.Name)
		}
	}

	if service.Status != "active" && service.Status != "inactive" {
		return fmt.Errorf("invalid service status: must be 'active' or 'inactive'")
	}

	return nil
}

func (s *Service) GetService(ctx context.Context, clinicID, serviceID uuid.UUID) (*model.Service, error) {
	service, err := s.repo.GetService(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}
	return service, nil
}

func (s *Service) DeleteClinicStaff(ctx context.Context, clinicID uuid.UUID) error {
	return s.repo.DeleteClinicStaff(ctx, clinicID)
}
