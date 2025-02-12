package event

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventTracker struct {
	eventService EventService // Change from event.Service to EventService
}

func NewEventTracker(eventSvc EventService) *EventTracker {
	return &EventTracker{
		eventService: eventSvc,
	}
}

type EventTrackerMiddleware struct {
	eventService interface{} // Replace with actual service interface
}

// Create event context

func (m *EventTrackerMiddleware) TrackEvent(entityType, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create event context
		eventCtx := &EventContext{
			Resource:  entityType,
			Operation: action,
		}
		c.Set("eventCtx", eventCtx)

		// Process the request
		c.Next()

		// After request, create outbox event and emit to Redis
		if eventCtx.NewData != nil {
			payloadJSON, err := json.Marshal(eventCtx.NewData)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal event payload")
				return
			}

			// Create outbox event
			event := &OutboxEvent{
				ID:        uuid.New(),
				EventType: fmt.Sprintf("%s_%s", strings.ToUpper(entityType), strings.ToUpper(action)),
				Status:    "PENDING",
				Payload:   payloadJSON,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Save to outbox and emit to Redis
			if err := m.eventService.(EventService).CreateEvent(c.Request.Context(), event); err != nil {
				log.Error().Err(err).Msg("Failed to create event")
			}

			// Also emit directly
			if err := m.eventService.(EventService).Emit(EventType(event.EventType), map[string]interface{}{
				"id":      event.ID,
				"payload": string(event.Payload),
			}); err != nil {
				log.Error().Err(err).Msg("Failed to emit event")
			}
		}
	}
}

func NewEventTrackerMiddleware(svc interface{}) *EventTrackerMiddleware {
	return &EventTrackerMiddleware{
		eventService: svc,
	}
}

func (t *EventTracker) Track(resource, operation string) gin.HandlerFunc {
	return t.TrackEvent(resource, operation)
}

func (t *EventTracker) TrackEvent(resource, operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventCtx := &EventContext{
			Resource:  resource,
			Operation: operation,
		}
		c.Set("eventCtx", eventCtx)
		c.Next()
	}
}
