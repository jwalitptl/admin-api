package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type OutboxRepository interface {
	Create(ctx context.Context, event *model.OutboxEvent) error
	GetPendingEvents(ctx context.Context, limit int) ([]*model.OutboxEvent, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.OutboxStatus, err *string) error
	GetPendingEventsWithLock(ctx context.Context, limit int) ([]*model.OutboxEvent, error)
	BeginTx(ctx context.Context) (*sql.Tx, error)
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, status string, errorMessage *string, retryAt *time.Time) error
	MoveToDeadLetter(ctx context.Context, tx *sql.Tx, evt *model.OutboxEvent) error
	DeleteProcessedBefore(ctx context.Context, before time.Time) (int64, error)
}

type outboxRepository struct {
	BaseRepository
}

func NewOutboxRepository(base BaseRepository) repository.OutboxRepository {
	return &outboxRepository{base}
}

func (r *outboxRepository) Create(ctx context.Context, event *model.OutboxEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}
	if event.Payload == nil {
		return fmt.Errorf("event payload cannot be nil")
	}

	query := `
		INSERT INTO outbox_events (
			id, event_type, payload, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`
	event.ID = uuid.New()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	event.Status = "pending" // Set default status

	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.Status,
		event.CreatedAt,
		event.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}
	return nil
}

func (r *outboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*model.OutboxEvent, error) {
	query := `
		SELECT id, event_type, payload, status, created_at, updated_at, error_message
		FROM outbox_events
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	var events []*model.OutboxEvent
	err := r.db.SelectContext(ctx, &events, query, string(model.OutboxStatusPending), limit)
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

func (r *outboxRepository) GetPendingEventsWithLock(ctx context.Context, limit int) ([]*model.OutboxEvent, error) {
	query := `
		SELECT id, event_type, payload, status, error_message, created_at, processed_at, updated_at
		FROM outbox_events
		WHERE status IN ('pending', 'retry')
		AND (retry_at IS NULL OR retry_at <= NOW())
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`
	var events []*model.OutboxEvent
	err := r.db.SelectContext(ctx, &events, query, limit)
	return events, err
}

func (r *outboxRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *outboxRepository) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, status string, errorMessage *string, retryAt *time.Time) error {
	query := `
		UPDATE outbox_events
		SET status = $1, 
			error_message = $2,
			retry_at = $4,
			processed_at = CASE WHEN $1 = 'processed' THEN NOW() ELSE processed_at END,
			updated_at = NOW()
		WHERE id = $3
	`
	_, err := tx.ExecContext(ctx, query, status, errorMessage, id, retryAt)
	return err
}

func (r *outboxRepository) MoveToDeadLetter(ctx context.Context, tx *sql.Tx, evt *model.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events_deadletter (
			event_id, event_type, payload, error_message, 
			retry_count, last_retry_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := tx.ExecContext(ctx, query, evt.ID, evt.EventType, evt.Payload,
		evt.ErrorMessage, evt.RetryCount, evt.RetryAt)
	return err
}

func (r *outboxRepository) DeleteProcessedBefore(ctx context.Context, before time.Time) (int64, error) {
	query := `
		DELETE FROM outbox_events
		WHERE status = 'processed'
		AND processed_at < $1
	`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete processed events: %w", err)
	}

	return result.RowsAffected()
}
