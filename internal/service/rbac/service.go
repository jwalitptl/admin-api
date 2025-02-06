package rbac

import (
	"context"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

type Service interface {
	// Role methods
	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, organizationID *uuid.UUID) ([]*model.Role, error)

	// Permission methods
	CreatePermission(ctx context.Context, permission *model.Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error)
	UpdatePermission(ctx context.Context, permission *model.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]*model.Permission, error)

	// Role assignments
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
	RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error

	// Authorization
	HasPermission(ctx context.Context, clinicianID uuid.UUID, permissionName string, organizationID uuid.UUID) (bool, error)

	// New method
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error)

	// Additional method
	ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error)
}

type Repository interface {
	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, organizationID *uuid.UUID) ([]*model.Role, error)
	CreatePermission(ctx context.Context, permission *model.Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error)
	UpdatePermission(ctx context.Context, permission *model.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]*model.Permission, error)
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
	RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error
	HasPermission(ctx context.Context, clinicianID uuid.UUID, permissionName string, organizationID uuid.UUID) (bool, error)
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error)
	ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateRole(ctx context.Context, role *model.Role) error {
	return s.repo.CreateRole(ctx, role)
}

func (s *service) GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	return s.repo.GetRole(ctx, id)
}

func (s *service) UpdateRole(ctx context.Context, role *model.Role) error {
	return s.repo.UpdateRole(ctx, role)
}

func (s *service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *service) ListRoles(ctx context.Context, organizationID *uuid.UUID) ([]*model.Role, error) {
	return s.repo.ListRoles(ctx, organizationID)
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

func (s *service) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.repo.RemovePermissionFromRole(ctx, roleID, permissionID)
}

func (s *service) AssignRoleToClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error {
	return s.repo.AssignRoleToClinician(ctx, clinicianID, roleID, orgID)
}

func (s *service) RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, orgID uuid.UUID) error {
	return s.repo.RemoveRoleFromClinician(ctx, clinicianID, roleID, orgID)
}

func (s *service) HasPermission(ctx context.Context, clinicianID uuid.UUID, permissionName string, organizationID uuid.UUID) (bool, error) {
	return s.repo.HasPermission(ctx, clinicianID, permissionName, organizationID)
}

func (s *service) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error) {
	return s.repo.ListRolePermissions(ctx, roleID)
}

func (s *service) ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error) {
	return s.repo.ListClinicianRoles(ctx, clinicianID, orgID)
}
