package event

import (
	"context"

	"github.com/jwalitptl/admin-api/internal/model"
)

type EventType string

type EventService interface {
	CreateEvent(ctx context.Context, event *model.OutboxEvent) error
	Emit(ctx context.Context, eventType string, payload interface{}) error
}

func NewEventTrackerMiddleware(svc EventService) *EventTrackerMiddleware {
	return &EventTrackerMiddleware{
		eventService: svc,
	}
}

type EventTrackerMiddleware struct {
	eventService EventService
}
