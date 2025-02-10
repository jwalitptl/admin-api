package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all application metrics
type Metrics struct {
	// Outbox related metrics
	OutboxEventsProcessed   prometheus.Counter
	OutboxEventsFailed      prometheus.Counter
	OutboxProcessingLatency prometheus.Histogram
	OutboxQueueSize         prometheus.Gauge
	OutboxRetries           *prometheus.CounterVec

	// Database metrics
	DatabaseOperations  *prometheus.CounterVec
	DatabaseLatency     *prometheus.HistogramVec
	DatabaseConnections prometheus.Gauge

	// Redis metrics
	RedisOperations  *prometheus.CounterVec
	RedisLatency     *prometheus.HistogramVec
	RedisConnections prometheus.Gauge
}

// NewMetrics creates and registers all application metrics
func NewMetrics(namespace, subsystem string) *Metrics {
	return &Metrics{
		// Outbox metrics
		OutboxEventsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "outbox_events_processed_total",
			Help:      "Total number of successfully processed outbox events",
		}),
		OutboxEventsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "outbox_events_failed_total",
			Help:      "Total number of failed outbox events",
		}),
		OutboxProcessingLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "outbox_processing_duration_seconds",
			Help:      "Time spent processing outbox events",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),
		OutboxQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "outbox_queue_size",
			Help:      "Current number of events in the outbox queue",
		}),
		OutboxRetries: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "outbox_retry_attempts_total",
			Help:      "Total number of retry attempts for outbox events",
		}, []string{"event_type"}),

		// Database metrics
		DatabaseOperations: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "database_operations_total",
			Help:      "Total number of database operations",
		}, []string{"operation", "status"}),
		DatabaseLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "database_operation_duration_seconds",
			Help:      "Duration of database operations",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}, []string{"operation"}),
		DatabaseConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "database_connections",
			Help:      "Current number of database connections",
		}),

		// Redis metrics
		RedisOperations: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "redis_operations_total",
			Help:      "Total number of Redis operations",
		}, []string{"operation", "status"}),
		RedisLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "redis_operation_duration_seconds",
			Help:      "Duration of Redis operations",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5},
		}, []string{"operation"}),
		RedisConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "redis_connections",
			Help:      "Current number of Redis connections",
		}),
	}
}

func New(namespace string) *Metrics {
	return &Metrics{
		OutboxEventsProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_processed_total",
			Help:      "Total number of outbox events processed",
		}),
		OutboxEventsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_failed_total",
			Help:      "Total number of outbox events that failed processing",
		}),
		OutboxProcessingLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "processing_duration_seconds",
			Help:      "Time spent processing outbox events",
		}),
		DatabaseOperations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "database_operations_total",
			Help:      "Total number of database operations",
		}, []string{"operation", "status"}),
	}
}
