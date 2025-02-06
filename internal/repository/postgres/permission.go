package postgres

import (
	"context"

	"github.com/jwalitptl/admin-api/internal/model"

	"github.com/jmoiron/sqlx"
)

type PermissionRepository struct {
	db *sqlx.DB
}

func NewPermissionRepository(db *sqlx.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) List(ctx context.Context) ([]*model.Permission, error) {
	var permissions []*model.Permission
	query := `
		SELECT id, name, description, created_at, updated_at 
		FROM permissions 
		ORDER BY name`
	err := r.db.SelectContext(ctx, &permissions, query)
	return permissions, err
}
