package event

import (
	"context"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/pkg/messaging"
)

type Service struct {
	outboxRepo repository.OutboxRepository
	broker     messaging.MessageBroker
	auditor    *audit.Service
}

func NewService(outboxRepo repository.OutboxRepository, broker messaging.MessageBroker, auditor *audit.Service) *Service {
	return &Service{
		outboxRepo: outboxRepo,
		broker:     broker,
		auditor:    auditor,
	}
}

func (s *Service) CreateEvent(ctx context.Context, event *model.OutboxEvent) error {
	return s.outboxRepo.Create(ctx, event)
}

func (s *Service) Emit(eventType EventType, payload map[string]interface{}) error {
	// Implementation here
	return nil
}
