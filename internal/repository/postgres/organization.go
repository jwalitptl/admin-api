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

type organizationRepository struct {
	BaseRepository
}

func NewOrganizationRepository(base BaseRepository) repository.OrganizationRepository {
	return &organizationRepository{base}
}

// All organization repository methods here

func (r *organizationRepository) CreateOrganization(ctx context.Context, org *model.Organization) error {
	query := `
		INSERT INTO organizations (
			id, account_id, name, status, region_code,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			org.ID,
			org.AccountID,
			org.Name,
			org.Status,
			r.GetRegionFromContext(ctx),
			org.CreatedAt,
			org.UpdatedAt,
		)
		return err
	})
}

func (r *organizationRepository) GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	query := `
		SELECT * FROM organizations 
		WHERE id = $1 AND deleted_at IS NULL
	`
	var org model.Organization
	if err := r.GetDB().GetContext(ctx, &org, query, id); err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return &org, nil
}

func (r *organizationRepository) UpdateOrganization(ctx context.Context, org *model.Organization) error {
	query := `
		UPDATE organizations
		SET name = $1, status = $2, updated_at = $3
		WHERE id = $4
	`
	org.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		org.Name,
		org.Status,
		org.UpdatedAt,
		org.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

func (r *organizationRepository) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM organizations
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

func (r *organizationRepository) ListOrganizations(ctx context.Context, accountID uuid.UUID) ([]*model.Organization, error) {
	query := `
		SELECT id, account_id, name, status, created_at, updated_at
		FROM organizations
		WHERE account_id = $1
		ORDER BY created_at DESC
	`
	var orgs []*model.Organization
	err := r.db.SelectContext(ctx, &orgs, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return orgs, nil
}

// ... rest of organization methods from organization_impl.go
