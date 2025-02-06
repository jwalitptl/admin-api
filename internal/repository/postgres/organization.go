package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
)

// All organization repository methods here

func (r *organizationRepository) CreateOrganization(ctx context.Context, org *model.Organization) error {
	query := `
		INSERT INTO organizations (id, account_id, name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		org.ID,
		org.AccountID,
		org.Name,
		org.Status,
		org.CreatedAt,
		org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}
	return nil
}

func (r *organizationRepository) GetOrganization(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	query := `
		SELECT id, account_id, name, status, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`
	var org model.Organization
	err := r.db.GetContext(ctx, &org, query, id)
	if err != nil {
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
