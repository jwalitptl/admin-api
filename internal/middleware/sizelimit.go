package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SizeLimitConfig represents size limit configuration
type SizeLimitConfig struct {
	MaxBodySize   int64 // in bytes
	MaxUploadSize int64 // in bytes
	MaxHeaderSize int   // in bytes
	ErrorMessage  string
	SkipPaths     []string
}

func DefaultSizeLimitConfig() SizeLimitConfig {
	return SizeLimitConfig{
		MaxBodySize:   1 << 20,  // 1MB
		MaxUploadSize: 10 << 20, // 10MB
		MaxHeaderSize: 1 << 14,  // 16KB
		ErrorMessage:  "Request size exceeds limit",
	}
}

// SizeLimit middleware limits request sizes
func SizeLimit(config SizeLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip configured paths
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Check content length
		if c.Request.ContentLength > config.MaxBodySize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("%s: body size exceeds %d bytes",
					config.ErrorMessage, config.MaxBodySize),
			})
			return
		}

		// Check header size
		headerSize := 0
		for name, values := range c.Request.Header {
			headerSize += len(name)
			for _, value := range values {
				headerSize += len(value)
			}
		}

		if headerSize > config.MaxHeaderSize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("%s: header size exceeds %d bytes",
					config.ErrorMessage, config.MaxHeaderSize),
			})
			return
		}

		c.Next()
	}
}
