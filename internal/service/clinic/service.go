package clinic

import (
	"context"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

type Service interface {
	CreateClinic(ctx context.Context, clinic *model.Clinic) error
	GetClinic(ctx context.Context, id uuid.UUID) (*model.Clinic, error)
	UpdateClinic(ctx context.Context, clinic *model.Clinic) error
	DeleteClinic(ctx context.Context, id uuid.UUID) error
	ListClinics(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error)
}

type Repository interface {
	Create(ctx context.Context, clinic *model.Clinic) error
	Get(ctx context.Context, id uuid.UUID) (*model.Clinic, error)
	Update(ctx context.Context, clinic *model.Clinic) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateClinic(ctx context.Context, clinic *model.Clinic) error {
	return s.repo.Create(ctx, clinic)
}

func (s *service) GetClinic(ctx context.Context, id uuid.UUID) (*model.Clinic, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) UpdateClinic(ctx context.Context, clinic *model.Clinic) error {
	return s.repo.Update(ctx, clinic)
}

func (s *service) DeleteClinic(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) ListClinics(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error) {
	return s.repo.List(ctx, organizationID)
}
