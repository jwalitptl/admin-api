package circuitbreaker

import (
	"sync"
	"time"
)

type Settings struct {
	Name        string
	MaxRequests int
	Interval    time.Duration
	Timeout     time.Duration
}

type CircuitBreaker struct {
	name        string
	maxRequests int
	interval    time.Duration
	timeout     time.Duration
	failures    int
	lastFailure time.Time
	state       string
	mu          sync.RWMutex
}

func NewCircuitBreaker(settings Settings) *CircuitBreaker {
	return &CircuitBreaker{
		name:        settings.Name,
		maxRequests: settings.MaxRequests,
		interval:    settings.Interval,
		timeout:     settings.Timeout,
		state:       "closed",
	}
}
