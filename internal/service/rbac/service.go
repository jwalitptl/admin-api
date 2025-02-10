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

type Service interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, orgID uuid.UUID) ([]*model.Role, error)
	AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error
	RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error
	CreatePermission(ctx context.Context, permission *model.Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error)
	UpdatePermission(ctx context.Context, permission *model.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]*model.Permission, error)
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
	RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error)
	ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error)
}

type service struct {
	repo    repository.RBACRepository
	auditor *audit.Service
}

func NewService(repo repository.RBACRepository, auditor *audit.Service) Service {
	// Initialize system roles if they don't exist
	ctx := context.Background()
	for _, roleName := range []string{systemRoleAdmin, systemRoleUser} {
		role := &model.Role{
			Name:         roleName,
			IsSystemRole: true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		// Ignore error if role already exists
		_ = repo.CreateRole(ctx, role)
	}

	return &service{
		repo:    repo,
		auditor: auditor,
	}
}

func (s *service) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
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

func (s *service) CreateRole(ctx context.Context, role *model.Role) error {
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

func (s *service) GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error) {
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

func (s *service) UpdateRole(ctx context.Context, role *model.Role) error {
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

func (s *service) DeleteRole(ctx context.Context, id uuid.UUID) error {
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

func (s *service) ListRoles(ctx context.Context, orgID uuid.UUID) ([]*model.Role, error) {
	roles, err := s.repo.ListRoles(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

func (s *service) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
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

func (s *service) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
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

func (s *service) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permission string) error {
	if err := s.repo.AddPermissionToRole(ctx, roleID, permission); err != nil {
		return fmt.Errorf("failed to add permission to role: %w", err)
	}
	return nil
}

func (s *service) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	if err := s.repo.RemovePermissionFromRole(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}
	return nil
}

func (s *service) CreatePermission(ctx context.Context, permission *model.Permission) error {
	return s.repo.CreatePermission(ctx, permission)
}

func (s *service) GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error) {
	return s.repo.GetPermission(ctx, id)
}

func (s *service) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	return s.repo.UpdatePermission(ctx, permission)
}

func (s *service) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePermission(ctx, id)
}

func (s *service) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *service) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.repo.AssignPermissionToRole(ctx, roleID, permissionID)
}

func (s *service) AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error {
	return s.repo.AssignRoleToClinician(ctx, clinicianID, roleID, orgID)
}

func (s *service) RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error {
	return s.repo.RemoveRoleFromClinician(ctx, clinicianID, roleID, orgID)
}

func (s *service) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error) {
	return s.repo.ListRolePermissions(ctx, roleID)
}

func (s *service) ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error) {
	return s.repo.ListClinicianRoles(ctx, clinicianID, orgID)
}

func (s *service) validateRole(role *model.Role) error {
	if role.Name == "" {
		return fmt.Errorf("role name is required")
	}

	if role.Name == systemRoleAdmin || role.Name == systemRoleUser {
		role.IsSystemRole = true
	}

	if role.OrganizationID == nil && !role.IsSystemRole {
		return fmt.Errorf("organization ID is required for non-system roles")
	}

	return nil
}

func (s *service) getCurrentUserID(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}
