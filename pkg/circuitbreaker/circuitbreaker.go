package circuitbreaker

import (
	"fmt"
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

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.RLock()
	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = "half-open"
			cb.mu.Unlock()
		} else {
			cb.mu.RUnlock()
			return fmt.Errorf("circuit breaker is open")
		}
	} else {
		cb.mu.RUnlock()
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.maxRequests {
			cb.state = "open"
		}
		return err
	}

	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0
	return nil
}
