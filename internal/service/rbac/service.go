package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
)

const (
	systemRoleAdmin = "admin"
	systemRoleUser  = "user"
)

type Service struct {
	repo    repository.RBACRepository
	auditor *audit.Service
}

func NewService(repo repository.RBACRepository, auditor *audit.Service) *Service {
	return &Service{
		repo:    repo,
		auditor: auditor,
	}
}

func (s *Service) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	for _, role := range roles {
		perms, err := s.repo.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return false, fmt.Errorf("failed to get role permissions: %w", err)
		}

		for _, p := range perms {
			if p.Name == permission {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *Service) CreateRole(ctx context.Context, role *model.Role) error {
	if err := s.validateRole(role); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	role.ID = uuid.New()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	orgID := uuid.Nil
	if role.OrganizationID != nil {
		orgID = *role.OrganizationID
	}
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), orgID, "create", "role", role.ID, &audit.LogOptions{
		Changes: role,
	})

	return nil
}

func (s *Service) GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	role, err := s.repo.GetRole(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	orgID := uuid.Nil
	if role.OrganizationID != nil {
		orgID = *role.OrganizationID
	}
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), orgID, "read", "role", id, nil)
	return role, nil
}

func (s *Service) UpdateRole(ctx context.Context, role *model.Role) error {
	if err := s.validateRole(role); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	if role.IsSystemRole {
		return fmt.Errorf("cannot modify system roles")
	}

	role.UpdatedAt = time.Now()
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	orgID := uuid.Nil
	if role.OrganizationID != nil {
		orgID = *role.OrganizationID
	}
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), orgID, "update", "role", role.ID, &audit.LogOptions{
		Changes: role,
	})

	return nil
}

func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := s.repo.GetRole(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	if role.IsSystemRole {
		return fmt.Errorf("cannot delete system roles")
	}

	if err := s.repo.DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	orgID := uuid.Nil
	if role.OrganizationID != nil {
		orgID = *role.OrganizationID
	}
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), orgID, "delete", "role", id, nil)
	return nil
}

func (s *Service) ListRoles(ctx context.Context, orgID uuid.UUID) ([]*model.Role, error) {
	roles, err := s.repo.ListRoles(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

func (s *Service) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	if err := s.repo.AssignRoleToUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	orgID := uuid.Nil
	if role.OrganizationID != nil {
		orgID = *role.OrganizationID
	}
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), orgID, "assign_role", "user", userID, &audit.LogOptions{
		Changes: map[string]interface{}{
			"role_id": roleID,
		},
	})

	return nil
}

func (s *Service) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	if err := s.repo.RemoveRoleFromUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	orgID := uuid.Nil
	if role.OrganizationID != nil {
		orgID = *role.OrganizationID
	}
	s.auditor.Log(ctx, s.getCurrentUserID(ctx), orgID, "remove_role", "user", userID, &audit.LogOptions{
		Changes: map[string]interface{}{
			"role_id": roleID,
		},
	})

	return nil
}

func (s *Service) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	if err := s.repo.AddPermissionToRole(ctx, roleID, permission); err != nil {
		return fmt.Errorf("failed to add permission to role: %w", err)
	}
	return nil
}

func (s *Service) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	if err := s.repo.RemovePermissionFromRole(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}
	return nil
}

func (s *Service) CreatePermission(ctx context.Context, permission *model.Permission) error {
	return s.repo.CreatePermission(ctx, permission)
}

func (s *Service) GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error) {
	return s.repo.GetPermission(ctx, id)
}

func (s *Service) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	return s.repo.UpdatePermission(ctx, permission)
}

func (s *Service) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePermission(ctx, id)
}

func (s *Service) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *Service) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.repo.AssignPermissionToRole(ctx, roleID, permissionID)
}

func (s *Service) AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error {
	return s.repo.AssignRoleToClinician(ctx, clinicianID, roleID, orgID)
}

func (s *Service) RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error {
	return s.repo.RemoveRoleFromClinician(ctx, clinicianID, roleID, orgID)
}

func (s *Service) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error) {
	return s.repo.ListRolePermissions(ctx, roleID)
}

func (s *Service) ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error) {
	return s.repo.ListClinicianRoles(ctx, clinicianID, orgID)
}

func (s *Service) validateRole(role *model.Role) error {
	if role.Name == "" {
		return fmt.Errorf("role name is required")
	}

	if role.OrganizationID == nil && !role.IsSystemRole {
		return fmt.Errorf("organization ID is required for non-system roles")
	}

	return nil
}

func (s *Service) getCurrentUserID(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}
