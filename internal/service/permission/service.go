package permission

import (
	"context"
	"fmt"
	"time"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"

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
	repo    repository.PermissionRepository
	auditor *audit.Service
}

func NewService(repo repository.PermissionRepository, auditor *audit.Service) *Service {
	return &Service{
		repo:    repo,
		auditor: auditor,
	}
}

func (s *Service) ListPermissions(ctx context.Context, orgID uuid.UUID) ([]*model.Permission, error) {
	permissions, err := s.repo.List(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	return permissions, nil
}

func (s *Service) CreatePermission(ctx context.Context, permission *model.Permission) error {
	if err := s.validatePermission(permission); err != nil {
		return fmt.Errorf("invalid permission: %w", err)
	}

	permission.ID = uuid.New()
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, permission); err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), permission.OrganizationID, "create", "permission", permission.ID, &audit.LogOptions{
		Changes: permission,
	})

	return nil
}

func (s *Service) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	if err := s.validatePermission(permission); err != nil {
		return fmt.Errorf("invalid permission: %w", err)
	}

	permission.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, permission); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), permission.OrganizationID, "update", "permission", permission.ID, &audit.LogOptions{
		Changes: permission,
	})

	return nil
}

func (s *Service) DeletePermission(ctx context.Context, id uuid.UUID) error {
	permission, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get permission: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), permission.OrganizationID, "delete", "permission", id, nil)
	return nil
}

func (s *Service) GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error) {
	permission, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	s.auditor.Log(ctx, s.getCurrentUserID(ctx), permission.OrganizationID, "read", "permission", id, nil)
	return permission, nil
}

func (s *Service) validatePermission(permission *model.Permission) error {
	if permission.Name == "" {
		return fmt.Errorf("permission name is required")
	}

	if permission.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization ID is required")
	}

	if permission.Resource == "" {
		return fmt.Errorf("resource is required")
	}

	if permission.Action == "" {
		return fmt.Errorf("action is required")
	}

	return nil
}

func (s *Service) getCurrentUserID(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}
