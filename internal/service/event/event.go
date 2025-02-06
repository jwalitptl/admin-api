package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
)

type EventType string

const (
	PatientCreated EventType = "PATIENT_CREATED"
	PatientUpdated EventType = "PATIENT_UPDATED"
	PatientDeleted EventType = "PATIENT_DELETED"
)

type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

type Service interface {
	Emit(eventType EventType, payload map[string]interface{})
}

type service struct {
	outboxRepo postgres.OutboxRepository
}

func NewService(outboxRepo postgres.OutboxRepository) Service {
	return &service{
		outboxRepo: outboxRepo,
	}
}

func (s *service) Emit(eventType EventType, payload map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	// Marshal the event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal event")
		return
	}

	// Create outbox event
	outboxEvent := &model.OutboxEvent{
		EventType: string(eventType),
		Payload:   eventJSON,
	}

	// Store in outbox
	if err := s.outboxRepo.Create(context.Background(), outboxEvent); err != nil {
		log.Error().Err(err).Msg("Failed to store event in outbox")
		return
	}

	log.Info().
		Str("event_id", outboxEvent.ID.String()).
		Str("event_type", string(eventType)).
		Msg("Event stored in outbox")
}
