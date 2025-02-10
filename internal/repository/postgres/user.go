package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type userRepository struct {
	BaseRepository
}

func NewUserRepository(base BaseRepository) repository.UserRepository {
	return &userRepository{base}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (
			id, organization_id, email, password_hash, first_name,
			last_name, type, status, created_at, updated_at, region_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			user.ID,
			user.OrganizationID,
			user.Email,
			user.PasswordHash,
			user.FirstName,
			user.LastName,
			user.Type,
			user.Status,
			user.CreatedAt,
			user.UpdatedAt,
			r.GetRegionFromContext(ctx),
		)
		return err
	})
}

func (r *userRepository) Get(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT * FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user model.User
	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT * FROM users 
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user model.User
	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET
			organization_id = $1,
			email = $2,
			password_hash = $3,
			first_name = $4,
			last_name = $5,
			type = $6,
			status = $7,
			updated_at = $8
		WHERE id = $9 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		user.OrganizationID,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Type,
		user.Status,
		time.Now(),
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users 
		SET deleted_at = NOW() 
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *userRepository) List(ctx context.Context, filters *model.UserFilters) ([]*model.User, error) {
	query := `
		SELECT * FROM users 
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if filters.OrganizationID != uuid.Nil {
		query += fmt.Sprintf(" AND organization_id = $%d", len(args)+1)
		args = append(args, filters.OrganizationID)
	}

	if filters.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, filters.Type)
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, filters.Status)
	}

	query += " ORDER BY created_at DESC"

	var users []*model.User
	if err := r.db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

func (r *userRepository) AssignToClinic(ctx context.Context, userID, clinicID uuid.UUID) error {
	query := `
		INSERT INTO user_clinics (user_id, clinic_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, clinic_id) DO NOTHING
	`
	result, err := r.db.ExecContext(ctx, query, userID, clinicID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign user to clinic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user already assigned to clinic")
	}
	return nil
}

func (r *userRepository) RemoveFromClinic(ctx context.Context, userID, clinicID uuid.UUID) error {
	query := `
		DELETE FROM user_clinics
		WHERE user_id = $1 AND clinic_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, userID, clinicID)
	if err != nil {
		return fmt.Errorf("failed to remove user from clinic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not assigned to clinic")
	}
	return nil
}

func (r *userRepository) ListUserClinics(ctx context.Context, userID uuid.UUID) ([]*model.Clinic, error) {
	query := `
		SELECT c.id, c.name, c.location, c.status,
			   c.created_at, c.updated_at
		FROM clinics c
		JOIN user_clinics uc ON c.id = uc.clinic_id
		WHERE uc.user_id = $1
		ORDER BY c.name
	`
	var clinics []*model.Clinic
	err := r.db.SelectContext(ctx, &clinics, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user clinics: %w", err)
	}
	return clinics, nil
}

func (r *userRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	result, err := r.db.ExecContext(ctx, query, userID, roleID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role already assigned to user")
	}
	return nil
}

func (r *userRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not assigned to user")
	}
	return nil
}

func (r *userRepository) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.is_system_role,
			   r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`
	var roles []*model.Role
	err := r.db.SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user roles: %w", err)
	}
	return roles, nil
}

func (r *userRepository) UpdateEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error {
	query := `
		UPDATE users 
		SET email_verified = $1, updated_at = NOW() 
		WHERE id = $2
	`
	result, err := r.db.ExecContext(ctx, query, verified, userID)
	if err != nil {
		return fmt.Errorf("failed to update email verification status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
