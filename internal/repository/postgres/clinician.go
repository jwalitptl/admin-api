package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/jwalitptl/admin-api/internal/model"
)

// All clinician repository methods here

func (r *clinicianRepository) CreateClinician(ctx context.Context, clinician *model.Clinician) error {
	query := `
		INSERT INTO clinicians (id, email, name, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	clinician.ID = uuid.New()
	clinician.CreatedAt = time.Now()
	clinician.UpdatedAt = time.Now()

	// Log the password before hashing (for debugging)
	fmt.Printf("Password before hash: %s\n", clinician.Password)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(clinician.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	clinician.PasswordHash = string(hashedPassword)

	// Log the hashed password (for debugging)
	fmt.Printf("Password after hash: %s\n", clinician.PasswordHash)

	_, err = r.db.ExecContext(ctx, query,
		clinician.ID,
		clinician.Email,
		clinician.Name,
		clinician.PasswordHash, // Make sure we're using PasswordHash here
		clinician.Status,
		clinician.CreatedAt,
		clinician.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create clinician: %w", err)
	}
	return nil
}

func (r *clinicianRepository) GetClinician(ctx context.Context, id uuid.UUID) (*model.Clinician, error) {
	query := `
		SELECT id, email, name, password_hash, status, created_at, updated_at
		FROM clinicians
		WHERE id = $1
	`
	var clinician model.Clinician
	err := r.db.GetContext(ctx, &clinician, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician: %w", err)
	}
	return &clinician, nil
}

func (r *clinicianRepository) UpdateClinician(ctx context.Context, clinician *model.Clinician) error {
	query := `
		UPDATE clinicians
		SET name = $1, email = $2, status = $3, updated_at = $4
		WHERE id = $5
	`
	clinician.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		clinician.Name,
		clinician.Email,
		clinician.Status,
		clinician.UpdatedAt,
		clinician.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update clinician: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinician not found")
	}

	return nil
}

func (r *clinicianRepository) DeleteClinician(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM clinicians
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete clinician: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinician not found")
	}

	return nil
}

func (r *clinicianRepository) ListClinicians(ctx context.Context) ([]*model.Clinician, error) {
	query := `
		SELECT id, email, name, status, created_at, updated_at
		FROM clinicians
		ORDER BY created_at DESC
	`
	var clinicians []*model.Clinician
	err := r.db.SelectContext(ctx, &clinicians, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list clinicians: %w", err)
	}
	return clinicians, nil
}

func (r *clinicianRepository) GetByEmail(ctx context.Context, email string) (*model.Clinician, error) {
	query := `
		SELECT id, email, name, password_hash, status, created_at, updated_at
		FROM clinicians
		WHERE email = $1
	`
	var clinician model.Clinician
	err := r.db.GetContext(ctx, &clinician, query, email)
	if err != nil {
		fmt.Printf("Error getting clinician by email: %v\n", err)
		return nil, fmt.Errorf("failed to get clinician by email: %w", err)
	}

	fmt.Printf("Found clinician: %+v\n", clinician)
	fmt.Printf("Password hash length: %d\n", len(clinician.PasswordHash))
	return &clinician, nil
}

func (r *clinicianRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	query := `
		UPDATE clinicians
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`
	result, err := r.db.ExecContext(ctx, query,
		hashedPassword,
		time.Now(),
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinician not found")
	}

	return nil
}

func (r *clinicianRepository) VerifyPassword(ctx context.Context, email, password string) (*model.Clinician, error) {
	clinician, err := r.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinician: %w", err)
	}

	fmt.Printf("Verifying password:\n")
	fmt.Printf("Input password: %s\n", password)
	fmt.Printf("Stored hash: %s\n", clinician.PasswordHash)

	err = bcrypt.CompareHashAndPassword([]byte(clinician.PasswordHash), []byte(password))
	if err != nil {
		fmt.Printf("Password comparison failed: %v\n", err)
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	return clinician, nil
}

func (r *clinicianRepository) AssignToClinic(ctx context.Context, clinicianID, clinicID uuid.UUID) error {
	// Debug logging
	fmt.Printf("Attempting to assign clinician %s to clinic %s\n", clinicianID, clinicID)

	// First verify with direct queries
	var clinicCount, clinicianCount int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM clinics WHERE id = $1", clinicID).Scan(&clinicCount)
	if err != nil {
		return fmt.Errorf("failed to check clinic: %w", err)
	}
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM clinicians WHERE id = $1", clinicianID).Scan(&clinicianCount)
	if err != nil {
		return fmt.Errorf("failed to check clinician: %w", err)
	}

	fmt.Printf("Direct count - Clinic: %d, Clinician: %d\n", clinicCount, clinicianCount)

	if clinicCount == 0 {
		return fmt.Errorf("clinic not found")
	}
	if clinicianCount == 0 {
		return fmt.Errorf("clinician not found")
	}

	query := `
		INSERT INTO clinic_clinicians (clinic_id, clinician_id)
		VALUES ($1, $2)
		ON CONFLICT (clinic_id, clinician_id) DO NOTHING
	`
	result, err := r.db.ExecContext(ctx, query, clinicID, clinicianID)
	if err != nil {
		return fmt.Errorf("failed to assign clinician to clinic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinician already assigned to clinic")
	}

	return nil
}

func (r *clinicianRepository) ListClinicClinicians(ctx context.Context, clinicID uuid.UUID) ([]*model.Clinician, error) {
	query := `
		SELECT c.id, c.email, c.name, c.status, c.created_at, c.updated_at
		FROM clinicians c
		JOIN clinic_clinicians cc ON c.id = cc.clinician_id
		WHERE cc.clinic_id = $1
		ORDER BY c.created_at DESC
	`
	var clinicians []*model.Clinician
	err := r.db.SelectContext(ctx, &clinicians, query, clinicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clinic clinicians: %w", err)
	}
	return clinicians, nil
}

func (r *clinicianRepository) ListClinicianClinics(ctx context.Context, clinicianID uuid.UUID) ([]*model.Clinic, error) {
	query := `
		SELECT c.id, c.organization_id, c.name, c.location, c.status, c.created_at, c.updated_at
		FROM clinics c
		JOIN clinic_clinicians cc ON c.id = cc.clinic_id
		WHERE cc.clinician_id = $1
		ORDER BY c.created_at DESC
	`
	var clinics []*model.Clinic
	err := r.db.SelectContext(ctx, &clinics, query, clinicianID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clinician clinics: %w", err)
	}
	return clinics, nil
}

func (r *clinicianRepository) RemoveFromClinic(ctx context.Context, clinicianID, clinicID uuid.UUID) error {
	query := `
		DELETE FROM clinic_clinicians
		WHERE clinician_id = $1 AND clinic_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, clinicianID, clinicID)
	if err != nil {
		return fmt.Errorf("failed to remove clinician from clinic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinician not assigned to clinic")
	}
	return nil
}

func (r *clinicianRepository) AssignRole(ctx context.Context, clinicianID, roleID uuid.UUID) error {
	query := `
		INSERT INTO clinician_roles (clinician_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (clinician_id, role_id) DO NOTHING
	`
	result, err := r.db.ExecContext(ctx, query, clinicianID, roleID)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role already assigned or not found")
	}
	return nil
}

func (r *clinicianRepository) RemoveRole(ctx context.Context, clinicianID, roleID uuid.UUID) error {
	query := `DELETE FROM clinician_roles WHERE clinician_id = $1 AND role_id = $2`
	result, err := r.db.ExecContext(ctx, query, clinicianID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("role not assigned")
	}
	return nil
}

func (r *clinicianRepository) ListClinicianRoles(ctx context.Context, clinicianID uuid.UUID) ([]*model.Role, error) {
	query := `
		SELECT r.* FROM roles r
		JOIN clinician_roles cr ON r.id = cr.role_id
		WHERE cr.clinician_id = $1
	`
	var roles []*model.Role
	if err := r.db.SelectContext(ctx, &roles, query, clinicianID); err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

func (r *clinicianRepository) GetRole(ctx context.Context, roleID uuid.UUID) (*model.Role, error) {
	query := `
		SELECT id, name, description, organization_id, is_system_role, created_at, updated_at 
		FROM roles 
		WHERE id = $1
	`
	var role model.Role
	err := r.db.GetContext(ctx, &role, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

func (r *clinicianRepository) AssignRoleToClinician(ctx context.Context, clinicianID, roleID, organizationID uuid.UUID) error {
	query := `
		INSERT INTO clinician_roles (clinician_id, role_id, organization_id)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, clinicianID, roleID, organizationID)
	return err
}
