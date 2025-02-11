package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/config"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/pkg/logger"
	"github.com/jwalitptl/admin-api/pkg/messaging"
	"github.com/jwalitptl/admin-api/pkg/messaging/redis"
	"github.com/jwalitptl/admin-api/pkg/metrics"
	"github.com/jwalitptl/admin-api/pkg/worker"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	logger     *logger.Logger
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
	processingLatency  *prometheus.HistogramVec
	retryCount         *prometheus.CounterVec
}

func NewEventWorker(outboxRepo postgres.OutboxRepository, broker messaging.Broker, logger *logger.Logger) *EventWorker {
	workerID := fmt.Sprintf("worker-%s", generateWorkerID())
	return &EventWorker{
		outboxRepo: outboxRepo,
		broker:     broker,
		logger:     logger.WithFields(map[string]interface{}{"worker_id": workerID}),
		batchSize:  100,
		workerID:   workerID,
		maxRetries: 3,
		retryDelay: 5 * time.Second,
		metrics: &WorkerMetrics{
			processedEvents:    processedEvents,
			failedEvents:       failedEvents,
			processingDuration: processingDuration,
			processingLatency:  eventProcessingLatency,
			retryCount:         eventRetryCount,
		},
	}
}

func setupHealthCheck(logger *logger.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		if err := http.ListenAndServe(":8081", mux); err != nil {
			logger.ZL.Error().Err(err).Msg("Health check server failed")
			os.Exit(1)
		}
	}()
}

func main() {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	// Initialize logger
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger := &logger.Logger{ZL: log.Logger}

	// Initialize database
	dbURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost",
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		"admin_db",
		cfg.Database.SSLMode,
	)
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		logger.ZL.Fatal().Err(err).Msg("Failed to connect to database")
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Redis broker
	broker, err := redis.NewRedisBroker(cfg.ToBrokerConfig(), &log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Redis broker")
	}
	defer broker.Close()

	// Initialize repositories
	baseRepo := postgres.NewBaseRepository(db)
	outboxRepo := postgres.NewOutboxRepository(baseRepo)

	// Initialize and start outbox processor
	processor := worker.NewOutboxProcessor(
		outboxRepo,
		broker,
		cfg.Outbox.ToWorkerConfig(),
		logger,
		metrics.New("outbox_processor"),
	)

	// Setup health check endpoints
	setupHealthCheck(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.ZL.Info().Msg("Shutting down...")
		cancel()
	}()

	processor.Start(ctx)
}

func (w *EventWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	w.logger.ZL.Info().Str("worker_id", w.workerID).Msg("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.ZL.Info().Str("worker_id", w.workerID).Msg("Worker shutting down")
			return
		case <-ticker.C:
			if err := w.processEvents(ctx); err != nil {
				w.logger.ZL.Error().Err(err).Str("worker_id", w.workerID).Msg("Error processing events")
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
					w.metrics.retryCount.WithLabelValues(evt.EventType).Inc()
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

				w.logger.ZL.Warn().Str("event_id", evt.ID.String()).Int("attempt", attempt+1).Err(publishErr).Msg("Retry publishing event")
			}

			if publishErr != nil {
				w.metrics.failedEvents.Inc()
				w.logger.ZL.Error().Str("event_id", evt.ID.String()).Err(publishErr).Msg("Failed to publish event after retries")

				errMsg := publishErr.Error()
				retryAt := time.Now().Add(w.retryDelay * time.Duration(w.maxRetries))
				if updateErr := w.outboxRepo.UpdateStatusTx(ctx, tx, evt.ID, "retry", &errMsg, &retryAt); updateErr != nil {
					w.logger.ZL.Error().Str("event_id", evt.ID.String()).Err(updateErr).Msg("Failed to update event status")
				}
				continue
			}

			if err := w.outboxRepo.UpdateStatusTx(ctx, tx, evt.ID, "processed", nil, nil); err != nil {
				w.logger.ZL.Error().Str("event_id", evt.ID.String()).Err(err).Msg("Failed to mark event as processed")
				continue
			}

			w.metrics.processedEvents.Inc()

			latency := time.Since(evt.CreatedAt).Seconds()
			w.metrics.processingLatency.WithLabelValues(evt.EventType).Observe(latency)
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
