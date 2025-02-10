package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig represents timeout middleware configuration
type TimeoutConfig struct {
	Duration time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Duration: 30 * time.Second,
	}
}

// Timeout adds request timeout
func Timeout(config TimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create timeout context
		ctx, cancel := context.WithTimeout(c.Request.Context(), config.Duration)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		done := make(chan bool, 1)
		go func() {
			c.Next()
			done <- true
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, ErrorResponse{
					Code:    http.StatusGatewayTimeout,
					Message: "Request timeout",
					TraceID: c.GetString(ContextRequestID),
				})
			}
		}
	}
}
