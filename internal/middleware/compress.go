package middleware

import (
	"compress/gzip"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

// CompressConfig represents compression configuration
type CompressConfig struct {
	Level     int
	MinLength int
	Types     []string
	Blacklist []string
}

// DefaultCompressConfig returns default compression configuration
func DefaultCompressConfig() CompressConfig {
	return CompressConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 1024,
		Types: []string{
			"application/json",
			"application/javascript",
			"text/css",
			"text/html",
			"text/plain",
			"text/xml",
		},
		Blacklist: []string{
			"/api/health",
			"/metrics",
		},
	}
}

// Compress adds gzip compression to responses
func Compress(config CompressConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip compression for blacklisted paths
		for _, path := range config.Blacklist {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Check if client accepts gzip
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Check content type
		contentType := c.Writer.Header().Get("Content-Type")
		shouldCompress := false
		for _, t := range config.Types {
			if strings.Contains(contentType, t) {
				shouldCompress = true
				break
			}
		}

		if !shouldCompress {
			c.Next()
			return
		}

		gz, err := gzip.NewWriterLevel(c.Writer, config.Level)
		if err != nil {
			c.Next()
			return
		}
		defer gz.Close()

		c.Writer = &gzipWriter{c.Writer, gz}
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Next()
	}
}
