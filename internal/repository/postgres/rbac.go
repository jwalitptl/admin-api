package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

// All RBAC repository methods here

func (r *rbacRepository) CreateRole(ctx context.Context, role *model.Role) error {
	query := `
		INSERT INTO roles (id, name, description, is_system_role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	role.ID = uuid.New()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.IsSystemRole,
		role.CreatedAt,
		role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (r *rbacRepository) GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	query := `
		SELECT id, name, description, is_system_role, created_at, updated_at
		FROM roles
		WHERE id = $1
	`
	var role model.Role
	err := r.db.GetContext(ctx, &role, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

func (r *rbacRepository) UpdateRole(ctx context.Context, role *model.Role) error {
	query := `
		UPDATE roles
		SET name = $1, description = $2, is_system_role = $3, updated_at = $4
		WHERE id = $5
	`
	role.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		role.Name,
		role.Description,
		role.IsSystemRole,
		role.UpdatedAt,
		role.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not found")
	}

	return nil
}

func (r *rbacRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM roles
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not found")
	}

	return nil
}

func (r *rbacRepository) ListRoles(ctx context.Context, organizationID *uuid.UUID) ([]*model.Role, error) {
	var query string
	var args []interface{}

	if organizationID != nil {
		query = `
			SELECT r.id, r.name, r.description, r.is_system_role, r.created_at, r.updated_at
			FROM roles r
			JOIN organization_roles or ON r.id = or.role_id
			WHERE or.organization_id = $1
			ORDER BY r.created_at DESC
		`
		args = append(args, *organizationID)
	} else {
		query = `
			SELECT id, name, description, is_system_role, created_at, updated_at
			FROM roles
			ORDER BY created_at DESC
		`
	}

	var roles []*model.Role
	err := r.db.SelectContext(ctx, &roles, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

func (r *rbacRepository) CreatePermission(ctx context.Context, permission *model.Permission) error {
	query := `
		INSERT INTO permissions (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	permission.ID = uuid.New()
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		permission.ID,
		permission.Name,
		permission.Description,
		permission.CreatedAt,
		permission.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}
	return nil
}

func (r *rbacRepository) GetPermission(ctx context.Context, id uuid.UUID) (*model.Permission, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM permissions
		WHERE id = $1
	`
	var permission model.Permission
	err := r.db.GetContext(ctx, &permission, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}
	return &permission, nil
}

func (r *rbacRepository) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	query := `
		UPDATE permissions
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`
	permission.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		permission.Name,
		permission.Description,
		permission.UpdatedAt,
		permission.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}

func (r *rbacRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM permissions
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}

func (r *rbacRepository) ListPermissions(ctx context.Context) ([]*model.Permission, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM permissions
		ORDER BY name ASC
	`
	var permissions []*model.Permission
	err := r.db.SelectContext(ctx, &permissions, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	return permissions, nil
}

func (r *rbacRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
	`
	_, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}
	return nil
}

func (r *rbacRepository) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `
		DELETE FROM role_permissions
		WHERE role_id = $1 AND permission_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("permission not assigned to role")
	}

	return nil
}

func (r *rbacRepository) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*model.Permission, error) {
	query := `
		SELECT p.id, p.name, p.description, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`
	var permissions []*model.Permission
	err := r.db.SelectContext(ctx, &permissions, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list role permissions: %w", err)
	}
	return permissions, nil
}

func (r *rbacRepository) AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error {
	query := `
		INSERT INTO clinician_roles (clinician_id, role_id, organization_id)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, clinicianID, roleID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to assign role to clinician: %w", err)
	}
	return nil
}

func (r *rbacRepository) RemoveRoleFromClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error {
	query := `
		DELETE FROM clinician_roles
		WHERE clinician_id = $1 AND role_id = $2 AND organization_id = $3
	`
	result, err := r.db.ExecContext(ctx, query, clinicianID, roleID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to remove role from clinician: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not assigned to clinician")
	}

	return nil
}

func (r *rbacRepository) ListClinicianRoles(ctx context.Context, clinicianID, orgID uuid.UUID) ([]*model.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.is_system_role, r.created_at, r.updated_at
		FROM roles r
		JOIN clinician_roles cr ON r.id = cr.role_id
		WHERE cr.clinician_id = $1 AND cr.organization_id = $2
		ORDER BY r.name ASC
	`
	var roles []*model.Role
	err := r.db.SelectContext(ctx, &roles, query, clinicianID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clinician roles: %w", err)
	}
	return roles, nil
}

func (r *rbacRepository) HasPermission(ctx context.Context, clinicianID uuid.UUID, permission string, organizationID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM clinician_roles cr
			JOIN role_permissions rp ON cr.role_id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			WHERE cr.clinician_id = $1
			AND cr.organization_id = $2
			AND p.name = $3
		)
	`
	var hasPermission bool
	err := r.db.GetContext(ctx, &hasPermission, query, clinicianID, organizationID, permission)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return hasPermission, nil
}

func (r *rbacRepository) AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error {
	return r.AssignRoleToClinician(ctx, clinicianID, roleID, uuid.Nil)
}

func (r *rbacRepository) RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error {
	return r.RemoveRoleFromClinician(ctx, clinicianID, roleID, uuid.Nil)
}
