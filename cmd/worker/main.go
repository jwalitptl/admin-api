package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jwalitptl/admin-api/config"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/pkg/logger"
	"github.com/jwalitptl/admin-api/pkg/messaging"
	"github.com/jwalitptl/admin-api/pkg/messaging/redis"
	"github.com/jwalitptl/admin-api/pkg/worker"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	processedEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_events_processed_total",
		Help: "The total number of processed outbox events",
	})
	failedEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_events_failed_total",
		Help: "The total number of failed outbox events",
	})
	processingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "event_processing_duration_seconds",
		Help:    "Time spent processing events",
		Buckets: prometheus.DefBuckets,
	})
	eventProcessingLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_processing_latency_seconds",
			Help:    "Time between event creation and processing",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"event_type"},
	)
	eventRetryCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "event_retry_total",
			Help: "Number of event retries",
		},
		[]string{"event_type"},
	)
)

type EventWorker struct {
	outboxRepo postgres.OutboxRepository
	broker     messaging.Broker
	logger     *zap.Logger
	batchSize  int
	workerID   string
	metrics    *WorkerMetrics
	lock       sync.Mutex
	maxRetries int
	retryDelay time.Duration
}

type WorkerMetrics struct {
	processedEvents    prometheus.Counter
	failedEvents       prometheus.Counter
	processingDuration prometheus.Histogram
}

func NewEventWorker(outboxRepo postgres.OutboxRepository, broker messaging.Broker, logger *zap.Logger) *EventWorker {
	workerID := fmt.Sprintf("worker-%s", generateWorkerID())
	return &EventWorker{
		outboxRepo: outboxRepo,
		broker:     broker,
		logger:     logger.With(zap.String("worker_id", workerID)),
		batchSize:  100,
		workerID:   workerID,
		maxRetries: 3,
		retryDelay: 5 * time.Second,
		metrics: &WorkerMetrics{
			processedEvents:    processedEvents,
			failedEvents:       failedEvents,
			processingDuration: processingDuration,
		},
	}
}

func setupHealthCheck(logger *zap.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		// Add DB and Redis health checks
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		if err := http.ListenAndServe(":8081", mux); err != nil {
			logger.Error("Health check server failed", zap.Error(err))
		}
	}()
}

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Initialize logger
	logger := logger.NewLogger(&logger.Config{
		Level:      logger.InfoLevel,
		TimeFormat: time.RFC3339,
		Output:     os.Stdout,
	})

	// Initialize Redis broker
	broker, err := redis.NewRedisBroker(cfg.Redis.ToBrokerConfig(), logger)
	if err != nil {
		logger.Fatal(err, "Failed to create Redis broker")
	}
	defer broker.Close()

	// Initialize repositories
	outboxRepo := postgres.NewOutboxRepository(db)

	// Initialize and start outbox processor
	processor := worker.NewOutboxProcessor(
		outboxRepo,
		broker,
		cfg.Outbox.ToWorkerConfig(),
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down...")
		cancel()
	}()

	processor.Start(ctx)
}

func (w *EventWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	w.logger.Info("Worker started", zap.String("worker_id", w.workerID))

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker shutting down", zap.String("worker_id", w.workerID))
			return
		case <-ticker.C:
			if err := w.processEvents(ctx); err != nil {
				w.logger.Error("Error processing events",
					zap.Error(err),
					zap.String("worker_id", w.workerID))
			}
		}
	}
}

func (w *EventWorker) processEvents(ctx context.Context) error {
	timer := prometheus.NewTimer(w.metrics.processingDuration)
	defer timer.ObserveDuration()

	events, err := w.outboxRepo.GetPendingEventsWithLock(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending events: %w", err)
	}

	tx, err := w.outboxRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Batch publish to Redis
	type eventBatch struct {
		events []*model.OutboxEvent
	}

	batches := make(map[string]*eventBatch)
	for _, evt := range events {
		key := evt.EventType
		if _, exists := batches[key]; !exists {
			batches[key] = &eventBatch{
				events: make([]*model.OutboxEvent, 0),
			}
		}
		batches[key].events = append(batches[key].events, evt)
	}

	// Process batches
	for _, batch := range batches {
		// Publish batch
		for _, evt := range batch.events {
			var publishErr error
			for attempt := 0; attempt < w.maxRetries; attempt++ {
				if attempt > 0 {
					backoff := time.Duration(attempt) * w.retryDelay
					time.Sleep(backoff)
				}

				w.lock.Lock()
				publishErr = w.broker.Publish(ctx, "events", map[string]interface{}{
					"type":    evt.EventType,
					"payload": evt.Payload,
				})
				w.lock.Unlock()

				if publishErr == nil {
					break
				}

				w.logger.Warn("Retry publishing event",
					zap.String("event_id", evt.ID.String()),
					zap.Int("attempt", attempt+1),
					zap.Error(publishErr))
			}

			if publishErr != nil {
				w.metrics.failedEvents.Inc()
				w.logger.Error("Failed to publish event after retries",
					zap.String("event_id", evt.ID.String()),
					zap.Error(publishErr))

				errMsg := publishErr.Error()
				retryAt := time.Now().Add(w.retryDelay * time.Duration(w.maxRetries))
				if updateErr := w.outboxRepo.UpdateStatusTx(ctx, tx, evt.ID, "retry", &errMsg, &retryAt); updateErr != nil {
					w.logger.Error("Failed to update event status",
						zap.String("event_id", evt.ID.String()),
						zap.Error(updateErr))
				}
				continue
			}

			if err := w.outboxRepo.UpdateStatusTx(ctx, tx, evt.ID, "processed", nil, nil); err != nil {
				w.logger.Error("Failed to mark event as processed",
					zap.String("event_id", evt.ID.String()),
					zap.Error(err))
				continue
			}

			w.metrics.processedEvents.Inc()
		}

		// Update statuses in transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

func generateWorkerID() string {
	// Generate a unique worker ID using hostname and timestamp
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())
}
