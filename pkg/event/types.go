package event

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EventType string

type EventContext struct {
	Resource   string
	Operation  string
	OldData    interface{}
	NewData    interface{}
	Additional map[string]interface{}
}

type OutboxEvent struct {
	ID           uuid.UUID  `json:"id"`
	EventType    string     `json:"event_type"`
	Payload      []byte     `json:"payload"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type EventService interface {
	CreateEvent(ctx context.Context, event *OutboxEvent) error
	Emit(eventType EventType, payload map[string]interface{}) error
}

type FieldExtractor interface {
	ExtractFields(obj interface{}, fields []string) map[string]interface{}
	ExtractChanges(old, new interface{}, fields []string) map[string]interface{}
}
