package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
)

// Copy the OutboxRepository interface here, but only expose what's needed by pkg/worker
type OutboxRepository interface {
	GetPendingEvents(ctx context.Context, limit int) ([]*model.OutboxEvent, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, err *string) error
}
