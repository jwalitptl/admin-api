package clinician

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

type Service interface {
	CreateClinician(ctx context.Context, clinician *model.Clinician) error
	GetClinician(ctx context.Context, id uuid.UUID) (*model.Clinician, error)
	UpdateClinician(ctx context.Context, clinician *model.Clinician) error
	DeleteClinician(ctx context.Context, id uuid.UUID) error
	ListClinicians(ctx context.Context) ([]*model.Clinician, error)

	// Clinic assignment
	AssignToClinic(ctx context.Context, clinicianID, clinicID uuid.UUID) error
	RemoveFromClinic(ctx context.Context, clinicID, clinicianID uuid.UUID) error
	ListClinicianClinics(ctx context.Context, clinicianID uuid.UUID) ([]*model.Clinic, error)
	ListClinicClinicians(ctx context.Context, clinicID uuid.UUID) ([]*model.Clinician, error)

	// Role management
	AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	ListClinicianRoles(ctx context.Context, clinicianID uuid.UUID) ([]*model.Role, error)
	GetRole(ctx context.Context, roleID uuid.UUID) (*model.Role, error)
	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error
}

type Repository interface {
	CreateClinician(ctx context.Context, clinician *model.Clinician) error
	GetClinician(ctx context.Context, id uuid.UUID) (*model.Clinician, error)
	UpdateClinician(ctx context.Context, clinician *model.Clinician) error
	DeleteClinician(ctx context.Context, id uuid.UUID) error
	ListClinicians(ctx context.Context) ([]*model.Clinician, error)
	AssignToClinic(ctx context.Context, clinicID, clinicianID uuid.UUID) error
	RemoveFromClinic(ctx context.Context, clinicID, clinicianID uuid.UUID) error
	ListClinicianClinics(ctx context.Context, clinicianID uuid.UUID) ([]*model.Clinic, error)
	ListClinicClinicians(ctx context.Context, clinicID uuid.UUID) ([]*model.Clinician, error)
	AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error
	ListClinicianRoles(ctx context.Context, clinicianID uuid.UUID) ([]*model.Role, error)
	GetRole(ctx context.Context, roleID uuid.UUID) (*model.Role, error)
	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateClinician(ctx context.Context, clinician *model.Clinician) error {
	return s.repo.CreateClinician(ctx, clinician)
}

func (s *service) GetClinician(ctx context.Context, id uuid.UUID) (*model.Clinician, error) {
	return s.repo.GetClinician(ctx, id)
}

func (s *service) UpdateClinician(ctx context.Context, clinician *model.Clinician) error {
	return s.repo.UpdateClinician(ctx, clinician)
}

func (s *service) DeleteClinician(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteClinician(ctx, id)
}

func (s *service) ListClinicians(ctx context.Context) ([]*model.Clinician, error) {
	return s.repo.ListClinicians(ctx)
}

func (s *service) AssignToClinic(ctx context.Context, clinicianID, clinicID uuid.UUID) error {
	fmt.Printf("Service - Assigning clinician %s to clinic %s\n", clinicianID, clinicID)

	// Note: Make sure parameter order matches repository
	return s.repo.AssignToClinic(ctx, clinicianID, clinicID)
}

func (s *service) RemoveFromClinic(ctx context.Context, clinicID, clinicianID uuid.UUID) error {
	return s.repo.RemoveFromClinic(ctx, clinicID, clinicianID)
}

func (s *service) ListClinicianClinics(ctx context.Context, clinicianID uuid.UUID) ([]*model.Clinic, error) {
	return s.repo.ListClinicianClinics(ctx, clinicianID)
}

func (s *service) ListClinicClinicians(ctx context.Context, clinicID uuid.UUID) ([]*model.Clinician, error) {
	return s.repo.ListClinicClinicians(ctx, clinicID)
}

func (s *service) AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error {
	return s.repo.AssignRole(ctx, clinicianID, roleID)
}

func (s *service) RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error {
	return s.repo.RemoveRole(ctx, clinicianID, roleID)
}

func (s *service) ListClinicianRoles(ctx context.Context, clinicianID uuid.UUID) ([]*model.Role, error) {
	return s.repo.ListClinicianRoles(ctx, clinicianID)
}

func (s *service) GetRole(ctx context.Context, roleID uuid.UUID) (*model.Role, error) {
	return s.repo.GetRole(ctx, roleID)
}

func (s *service) AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error {
	return s.repo.AssignRoleToClinician(ctx, clinicianID, roleID, organizationID)
}
