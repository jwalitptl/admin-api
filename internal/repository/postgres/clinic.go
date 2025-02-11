package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type clinicRepository struct {
	BaseRepository
}

func NewClinicRepository(base BaseRepository) repository.ClinicRepository {
	return &clinicRepository{base}
}

// All clinic repository methods here

func (r *clinicRepository) Create(ctx context.Context, clinic *model.Clinic) error {
	query := `
		INSERT INTO clinics (
			id, organization_id, name, location, status, region_code, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`
	clinic.ID = uuid.New()
	clinic.CreatedAt = time.Now()
	clinic.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		clinic.ID,
		clinic.OrganizationID,
		clinic.Name,
		clinic.Location,
		clinic.Status,
		clinic.RegionCode,
		clinic.CreatedAt,
		clinic.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create clinic: %w", err)
	}
	return nil
}

func (r *clinicRepository) Get(ctx context.Context, id uuid.UUID) (*model.Clinic, error) {
	query := `
		SELECT 
			id, organization_id, name, location, status, region_code, 
			created_at, updated_at
		FROM clinics
		WHERE id = $1
	`
	var clinic model.Clinic
	err := r.db.GetContext(ctx, &clinic, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinic: %w", err)
	}
	return &clinic, nil
}

func (r *clinicRepository) Update(ctx context.Context, clinic *model.Clinic) error {
	query := `
		UPDATE clinics
		SET name = $1, location = $2, status = $3, updated_at = $4
		WHERE id = $5
	`
	clinic.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		clinic.Name,
		clinic.Location,
		clinic.Status,
		clinic.UpdatedAt,
		clinic.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update clinic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinic not found")
	}

	return nil
}

func (r *clinicRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM clinics
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete clinic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("clinic not found")
	}

	return nil
}

func (r *clinicRepository) List(ctx context.Context, organizationID uuid.UUID) ([]*model.Clinic, error) {
	log.Printf("Debug - List clinics for org: %s", organizationID)
	query := `
		SELECT 
			id, organization_id, name, location, status, region_code, 
			created_at, updated_at
		FROM clinics
		WHERE organization_id = $1
		AND (COALESCE($2, '') = '' OR name ILIKE '%' || $2 || '%')
		AND (COALESCE($3, '') = '' OR status = $3)
		ORDER BY created_at DESC
	`
	var clinics []*model.Clinic
	search := ""
	if s := ctx.Value("search"); s != nil {
		search = s.(string)
	}
	status := ""
	if s := ctx.Value("status"); s != nil {
		status = s.(string)
	}

	log.Printf("Debug - Executing query with search='%s', status='%s'", search, status)
	err := r.db.SelectContext(ctx, &clinics, query, organizationID, search, status)
	if err != nil {
		log.Printf("Debug - Query error: %v", err)
		return nil, fmt.Errorf("failed to list clinics: %w", err)
	}
	log.Printf("Debug - Found %d clinics", len(clinics))
	return clinics, nil
}

func (r *clinicRepository) AssignStaff(ctx context.Context, staff *model.ClinicStaff) error {
	query := `INSERT INTO clinic_staff (clinic_id, user_id, role, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, staff.ClinicID, staff.UserID, staff.Role, staff.CreatedAt)
	return err
}

func (r *clinicRepository) ListStaff(ctx context.Context, clinicID uuid.UUID) ([]*model.ClinicStaff, error) {
	query := `
		SELECT 
			cs.clinic_id as "clinic_id",
			cs.user_id as "user_id",
			cs.role as "role",
			cs.created_at as "created_at"
		FROM clinic_staff cs
		WHERE cs.clinic_id = $1
	`
	var staff []*model.ClinicStaff
	err := r.db.SelectContext(ctx, &staff, query, clinicID)
	return staff, err
}

func (r *clinicRepository) RemoveStaff(ctx context.Context, clinicID, userID uuid.UUID) error {
	query := `DELETE FROM clinic_staff WHERE clinic_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, clinicID, userID)
	return err
}
