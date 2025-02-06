package permission

import (
	"context"

	"github.com/jwalitptl/admin-api/internal/model"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context) ([]*model.Permission, error)
	Create(ctx context.Context, permission *model.Permission) error
	Update(ctx context.Context, permission *model.Permission) error
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*model.Permission, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	return s.repo.List(ctx)
}

func (s *Service) CreatePermission(ctx context.Context, permission *model.Permission) error {
	return s.repo.Create(ctx, permission)
}

func (s *Service) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	return s.repo.Update(ctx, permission)
}

func (s *Service) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error) {
	return s.repo.Get(ctx, id)
}
