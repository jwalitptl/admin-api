package postgres

import (
	"context"
	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"

	"github.com/jmoiron/sqlx"
)

type PermissionRepository struct {
	db *sqlx.DB
}

func (r *PermissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	//TODO implement me
	panic("implement me")
}

func (r *PermissionRepository) Update(ctx context.Context, permission *model.Permission) error {
	//TODO implement me
	panic("implement me")
}

func (r *PermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	//TODO implement me
	panic("implement me")
}

func (r *PermissionRepository) Get(ctx context.Context, id uuid.UUID) (*model.Permission, error) {
	//TODO implement me
	panic("implement me")
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
