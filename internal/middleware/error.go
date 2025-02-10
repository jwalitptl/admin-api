package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id,omitempty"`
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only handle errors if they exist
		if len(c.Errors) == 0 {
			return
		}

		// Get trace ID from context
		traceID := c.GetString("trace_id")

		// Handle errors
		for _, e := range c.Errors {
			// Log error with context
			log.Error().
				Err(e.Err).
				Str("trace_id", traceID).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("client_ip", c.ClientIP()).
				Interface("meta", e.Meta).
				Msg("Request error")
		}

		// Return last error to client
		lastErr := c.Errors.Last()
		status := http.StatusInternalServerError

		// Check if it's a custom error type
		if err, ok := lastErr.Err.(interface{ StatusCode() int }); ok {
			status = err.StatusCode()
		}

		c.JSON(status, ErrorResponse{
			Code:    status,
			Message: lastErr.Error(),
			TraceID: traceID,
		})
	}
}
