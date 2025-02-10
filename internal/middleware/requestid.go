package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	HeaderXRequestID = "X-Request-ID"
	ContextRequestID = "request_id"
)

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID exists in header
		rid := c.GetHeader(HeaderXRequestID)
		if rid == "" {
			rid = uuid.New().String()
		}

		c.Set(ContextRequestID, rid)
		c.Header(HeaderXRequestID, rid)
		c.Next()
	}
}
