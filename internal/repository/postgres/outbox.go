package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/jwalitptl/admin-api/internal/model"
)

type OutboxRepository interface {
	Create(ctx context.Context, event *model.OutboxEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*model.OutboxEvent, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.OutboxStatus, err *string) error
}

type outboxRepository struct {
	db *sqlx.DB
}

func NewOutboxRepository(db *sqlx.DB) OutboxRepository {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) Create(ctx context.Context, event *model.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	event.ID = uuid.New()
	event.Status = model.OutboxStatusPending
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.Status,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return err
}

func (r *outboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*model.OutboxEvent, error) {
	query := `
		SELECT id, event_type, payload, status, created_at, updated_at, error
		FROM outbox_events
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	var events []*model.OutboxEvent
	err := r.db.SelectContext(ctx, &events, query, model.OutboxStatusPending, limit)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return events, err
}

func (r *outboxRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OutboxStatus, err *string) error {
	query := `
		UPDATE outbox_events
		SET status = $1, error = $2, updated_at = $3
		WHERE id = $4
	`

	_, dbErr := r.db.ExecContext(ctx, query, status, err, time.Now(), id)
	return dbErr
}
