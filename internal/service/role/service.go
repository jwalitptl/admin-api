package role

import (
	"context"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
)

type RoleService interface {
	CreateRole(ctx context.Context, role *model.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*model.Role, error)
	ListRoles(ctx context.Context, orgID uuid.UUID) ([]*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
}
