package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger returns a middleware that logs HTTP requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create custom response writer to capture response
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = w

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		logger := log.With().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("ip", c.ClientIP()).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Str("user_agent", c.Request.UserAgent())

		// Add request body for non-GET requests (optional, and be careful with sensitive data)
		if c.Request.Method != "GET" && len(requestBody) > 0 {
			logger = logger.Str("request", string(requestBody))
		}

		// Log based on status code
		logEvent := logger.Logger() // Create logger instance
		if c.Writer.Status() >= 500 {
			logEvent.Error().
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("ip", c.ClientIP()).
				Int("status", c.Writer.Status()).
				Dur("duration", duration).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Server error")
		} else if c.Writer.Status() >= 400 {
			logEvent.Warn().
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("ip", c.ClientIP()).
				Int("status", c.Writer.Status()).
				Dur("duration", duration).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Client error")
		} else {
			logEvent.Info().
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("ip", c.ClientIP()).
				Int("status", c.Writer.Status()).
				Dur("duration", duration).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Request processed")
		}
	}
}
