package event

import "github.com/gin-gonic/gin"

// EventHandler defines the interface for handlers that need event tracking
type EventHandler interface {
	RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *EventTracker)
}
