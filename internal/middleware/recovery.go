package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Recovery handles panics and logs them appropriately
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Log the error
				log.Error().
					Interface("error", err).
					Str("stack", string(stack)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Str("client_ip", c.ClientIP()).
					Str("request_id", c.GetString(ContextRequestID)).
					Msg("Request panic recovered")

				// Return error to client
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Code:    http.StatusInternalServerError,
					Message: "Internal server error",
					TraceID: c.GetString(ContextRequestID),
				})
			}
		}()
		c.Next()
	}
}
