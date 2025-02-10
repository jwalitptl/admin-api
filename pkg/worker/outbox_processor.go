package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"

	"github.com/jwalitptl/admin-api/pkg/logger"
	"github.com/jwalitptl/admin-api/pkg/messaging"
	"github.com/jwalitptl/admin-api/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Add configuration options
type OutboxProcessorConfig struct {
	BatchSize     int
	PollInterval  time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

type OutboxProcessor struct {
	repo    repository.OutboxRepository
	broker  messaging.Broker
	config  OutboxProcessorConfig
	logger  *logger.Logger
	metrics *metrics.Metrics
}

func NewOutboxProcessor(
	repo repository.OutboxRepository,
	broker messaging.Broker,
	config OutboxProcessorConfig,
	logger *logger.Logger,
	metrics *metrics.Metrics,
) *OutboxProcessor {
	// Config validation instead of defaults
	if config.BatchSize <= 0 {
		panic("BatchSize must be greater than 0")
	}
	if config.PollInterval <= 0 {
		panic("PollInterval must be greater than 0")
	}
	if config.RetryAttempts <= 0 {
		panic("RetryAttempts must be greater than 0")
	}
	if config.RetryDelay <= 0 {
		panic("RetryDelay must be greater than 0")
	}

	return &OutboxProcessor{
		repo:    repo,
		broker:  broker,
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
}

func (p *OutboxProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	p.logger.Info("Starting outbox processor")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Shutting down outbox processor")
			return
		case <-ticker.C:
			if err := p.processEvents(ctx); err != nil {
				p.logger.Error(err, "Failed to process events")
			}
		}
	}
}

func (p *OutboxProcessor) processEvents(ctx context.Context) error {
	timer := prometheus.NewTimer(p.metrics.OutboxProcessingLatency)
	defer timer.ObserveDuration()

	events, err := p.repo.GetPendingEventsWithLock(ctx, p.config.BatchSize)
	if err != nil {
		p.metrics.DatabaseOperations.WithLabelValues("get_pending_events", "error").Inc()
		return fmt.Errorf("failed to get pending events: %w", err)
	}
	p.metrics.DatabaseOperations.WithLabelValues("get_pending_events", "success").Inc()

	for _, event := range events {
		if err := p.processEvent(ctx, event); err != nil {
			p.logger.Error(err, "Failed to process event",
				"event_id", event.ID.String(),
				"event_type", event.EventType)
			continue
		}
	}

	return nil
}

func (p *OutboxProcessor) processEvent(ctx context.Context, event *model.OutboxEvent) error {
	err := retry(p.config.RetryAttempts, p.config.RetryDelay, func() error {
		return p.broker.Publish(ctx, event.EventType, event.Payload)
	})

	if err != nil {
		p.metrics.OutboxEventsFailed.Inc()
		errStr := err.Error()
		if updateErr := p.repo.UpdateStatusTx(ctx, nil, event.ID, string(model.OutboxStatusFailed), &errStr, nil); updateErr != nil {
			p.logger.Error(updateErr, "Failed to update event status")
		}
		return err
	}

	p.metrics.OutboxEventsProcessed.Inc()
	if err := p.repo.UpdateStatusTx(ctx, nil, event.ID, string(model.OutboxStatusProcessed), nil, nil); err != nil {
		p.logger.Error(err, "Failed to update event status", "event_id", event.ID.String())
		return err
	}

	return nil
}

// Helper retry function
func retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}
