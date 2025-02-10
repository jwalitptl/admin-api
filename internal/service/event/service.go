package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/pkg/messaging"
)

const (
	maxRetries  = 3
	retryDelay  = 5 * time.Second
	eventExpiry = 24 * time.Hour
)

type EventService struct {
	outboxRepo repository.OutboxRepository
	broker     messaging.Broker
	auditor    *audit.Service
}

func NewEventService(outboxRepo repository.OutboxRepository, broker messaging.Broker, auditor *audit.Service) *EventService {
	return &EventService{
		outboxRepo: outboxRepo,
		broker:     broker,
		auditor:    auditor,
	}
}

func (s *EventService) Emit(ctx context.Context, eventType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	event := &model.OutboxEvent{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   payloadJSON,
		Status:    string(model.OutboxStatusPending),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.outboxRepo.Create(ctx, event); err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, uuid.Nil, "emit", "event", event.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"event_type": eventType,
			"payload":    payload,
		},
	})

	// Try immediate processing
	go s.processEvent(ctx, event)

	return nil
}

func (s *EventService) ProcessPendingEvents(ctx context.Context) error {
	events, err := s.outboxRepo.GetPendingEventsWithLock(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to get pending events: %w", err)
	}

	for _, event := range events {
		if err := s.processEvent(ctx, event); err != nil {
			s.handleProcessingError(ctx, event, err)
		}
	}

	return nil
}

func (s *EventService) processEvent(ctx context.Context, event *model.OutboxEvent) error {
	// Start transaction
	tx, err := s.outboxRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Publish to message broker
	if err := s.broker.Publish(ctx, event.EventType, event.Payload); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// Update event status
	now := time.Now()
	if err := s.outboxRepo.UpdateStatusTx(ctx, tx, event.ID, string(model.OutboxStatusProcessed), nil, &now); err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, uuid.Nil, "process", "event", event.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"event_type": event.EventType,
			"status":     model.OutboxStatusProcessed,
		},
	})

	return nil
}

func (s *EventService) handleProcessingError(ctx context.Context, event *model.OutboxEvent, err error) {
	event.RetryCount++
	errMsg := err.Error()
	retryAt := time.Now().Add(retryDelay * time.Duration(event.RetryCount))

	if event.RetryCount >= maxRetries {
		// Move to dead letter queue
		if moveErr := s.outboxRepo.MoveToDeadLetter(ctx, nil, event); moveErr != nil {
			s.auditor.Log(ctx, uuid.Nil, uuid.Nil, "dead_letter_failed", "event", event.ID, &audit.LogOptions{
				Metadata: map[string]interface{}{
					"error": moveErr.Error(),
				},
			})
		}
		return
	}

	// Update status for retry
	if updateErr := s.outboxRepo.UpdateStatusTx(ctx, nil, event.ID, string(model.OutboxStatusFailed), &errMsg, &retryAt); updateErr != nil {
		s.auditor.Log(ctx, uuid.Nil, uuid.Nil, "retry_update_failed", "event", event.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": updateErr.Error(),
			},
		})
	}

	s.auditor.Log(ctx, uuid.Nil, uuid.Nil, "process_failed", "event", event.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"error":       err.Error(),
			"retry_count": event.RetryCount,
			"retry_at":    retryAt,
		},
	})
}

func (s *EventService) CleanupProcessedEvents(ctx context.Context) error {
	cutoff := time.Now().Add(-eventExpiry)
	count, err := s.outboxRepo.DeleteProcessedBefore(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup events: %w", err)
	}

	s.auditor.Log(ctx, uuid.Nil, uuid.Nil, "cleanup", "event", uuid.Nil, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"deleted_count": count,
			"cutoff":        cutoff,
		},
	})

	return nil
}
