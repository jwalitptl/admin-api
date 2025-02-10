package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

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

		// Log after request is processed
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// Log the request
		logger := log.With().
			Str("request_id", requestID).
			Str("client_ip", clientIP).
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("user_agent", c.Request.UserAgent())

		// Add request body for non-GET requests (optional, and be careful with sensitive data)
		if method != "GET" && len(requestBody) > 0 {
			logger = logger.Str("request", string(requestBody))
		}

		// Log based on status code
		logEvent := logger.Logger() // Create logger instance
		if statusCode >= 500 {
			logEvent.Error().
				Str("method", method).
				Str("path", path).
				Str("ip", clientIP).
				Int("status", statusCode).
				Dur("duration", latency).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Server error")
		} else if statusCode >= 400 {
			logEvent.Warn().
				Str("method", method).
				Str("path", path).
				Str("ip", clientIP).
				Int("status", statusCode).
				Dur("duration", latency).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Client error")
		} else {
			logEvent.Info().
				Str("method", method).
				Str("path", path).
				Str("ip", clientIP).
				Int("status", statusCode).
				Dur("duration", latency).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Request processed")
		}
	}
}
