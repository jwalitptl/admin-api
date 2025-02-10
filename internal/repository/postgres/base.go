package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/model"
)

// BaseRepository provides common functionality for all repositories
type BaseRepository struct {
	db *sqlx.DB
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *sqlx.DB) BaseRepository {
	return BaseRepository{db: db}
}

// GetDB returns the database instance
func (r *BaseRepository) GetDB() *sqlx.DB {
	return r.db
}

// WithTx executes a function within a transaction
func (r *BaseRepository) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// AddRegionFilter adds region filtering to queries
func (r *BaseRepository) AddRegionFilter(query string, regionCode string) string {
	return query + " AND region_code = ?"
}

// GetRegionFromContext gets region from context
func (r *BaseRepository) GetRegionFromContext(ctx context.Context) string {
	if regionCode, ok := ctx.Value("region_code").(string); ok {
		return regionCode
	}
	return "global"
}

// CreateAuditLog creates an audit log entry within a transaction
func (r *BaseRepository) CreateAuditLog(ctx context.Context, tx *sqlx.Tx, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, entity_type, entity_id, action, user_id,
			ip_address, changes, access_reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.ExecContext(ctx, query /* ... args ... */)
	return err
}
