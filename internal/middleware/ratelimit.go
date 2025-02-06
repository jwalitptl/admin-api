package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jwalitptl/admin-api/internal/handler"
)

type rateLimiter struct {
	sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) cleanup() {
	rl.Lock()
	defer rl.Unlock()

	now := time.Now()
	for ip, times := range rl.requests {
		var valid []time.Time
		for _, t := range times {
			if now.Sub(t) <= rl.window {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = valid
		}
	}
}

func (rl *rateLimiter) RateLimit() gin.HandlerFunc {
	go func() {
		for {
			time.Sleep(rl.window)
			rl.cleanup()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.Lock()
		now := time.Now()
		times := rl.requests[ip]

		// Remove old requests
		var valid []time.Time
		for _, t := range times {
			if now.Sub(t) <= rl.window {
				valid = append(valid, t)
			}
		}

		if len(valid) >= rl.limit {
			rl.Unlock()
			c.JSON(http.StatusTooManyRequests, handler.NewErrorResponse("rate limit exceeded"))
			c.Abort()
			return
		}

		rl.requests[ip] = append(valid, now)
		rl.Unlock()

		c.Next()
	}
}
