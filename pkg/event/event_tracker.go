package event

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

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
	eventService EventService
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
				log.Printf("Failed to marshal event payload: %v", err)
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
			if err := m.eventService.CreateEvent(c.Request.Context(), event); err != nil {
				log.Printf("Failed to create event: %v", err)
			}

			// Also emit directly
			if err := m.eventService.Emit(EventType(event.EventType), map[string]interface{}{
				"id":      event.ID,
				"payload": string(event.Payload),
			}); err != nil {
				log.Printf("Failed to emit event: %v", err)
			}
		}
	}
}

func NewEventTrackerMiddleware(eventSvc EventService) *EventTrackerMiddleware {
	return &EventTrackerMiddleware{
		eventService: eventSvc,
	}
}
