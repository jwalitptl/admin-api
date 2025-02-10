package appointment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/internal/service/appointment"
	"github.com/jwalitptl/admin-api/pkg/event"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Handler struct {
	service    *appointment.Service
	outboxRepo postgres.OutboxRepository
	validate   *validator.Validate
	metrics    *metrics
	limiter    *rate.Limiter
	breaker    *gobreaker.CircuitBreaker
	cache      *cache.Cache
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

type metrics struct {
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec
	cacheHits       *prometheus.CounterVec
	cacheMisses     *prometheus.CounterVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "appointment_request_duration_seconds",
				Help: "Duration of appointment requests in seconds",
			},
			[]string{"method", "endpoint", "status"},
		),
		requestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "appointment_requests_total",
				Help: "Total number of appointment requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		errorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "appointment_errors_total",
				Help: "Total number of appointment errors",
			},
			[]string{"method", "endpoint", "type"},
		),
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "appointment_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"type"},
		),
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "appointment_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"type"},
		),
	}
	reg.MustRegister(m.requestDuration, m.requestTotal, m.errorTotal, m.cacheHits, m.cacheMisses)
	return m
}

func NewHandler(service *appointment.Service, outboxRepo postgres.OutboxRepository) *Handler {
	// Allow 100 requests per second with burst of 200
	limiter := rate.NewLimiter(rate.Limit(100), 200)

	// Configure circuit breaker
	breakerSettings := gobreaker.Settings{
		Name:        "appointment-service",
		MaxRequests: 100,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Warn().
				Str("breaker", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("circuit breaker state changed")
		},
	}

	// Initialize cache
	c := cache.New(5*time.Minute, 10*time.Minute)

	// Initialize tracer and propagator
	tracer := otel.Tracer("appointment-handler")
	propagator := otel.GetTextMapPropagator()

	return &Handler{
		service:    service,
		outboxRepo: outboxRepo,
		validate:   validator.New(),
		metrics:    newMetrics(prometheus.DefaultRegisterer),
		limiter:    limiter,
		breaker:    gobreaker.NewCircuitBreaker(breakerSettings),
		cache:      c,
		tracer:     tracer,
		propagator: propagator,
	}
}

const (
	defaultTimeout = 10 * time.Second

	// Rate limiting
	requestsPerSecond = 100
	burstSize         = 200

	// Circuit breaker
	breakerMaxRequests = 100
	breakerInterval    = 10 * time.Second
	breakerTimeout     = 30 * time.Second
	breakerMinRequests = 10
	breakerFailRatio   = 0.6

	// Event types
	eventCreate = "APPOINTMENT_CREATE"
	eventUpdate = "APPOINTMENT_UPDATE"
	eventDelete = "APPOINTMENT_DELETE"

	// Error messages
	errInvalidID         = "invalid appointment ID"
	errInvalidClinicID   = "invalid clinic ID"
	errInvalidPatientID  = "invalid patient ID"
	errInvalidDate       = "invalid date format"
	errRequestTimeout    = "request timeout"
	errRateLimitExceeded = "rate limit exceeded"
	errCircuitOpen       = "service temporarily unavailable"

	// Cache settings
	cacheDuration        = 5 * time.Minute
	cacheCleanupInterval = 10 * time.Minute
	cacheKeyAppointment  = "appointment:%s"
	cacheKeyAvailability = "availability:%s:%s"
)

func (h *Handler) recordMetrics(start time.Time, method, endpoint, status string) {
	duration := time.Since(start).Seconds()
	h.metrics.requestDuration.WithLabelValues(method, endpoint, status).Observe(duration)
	h.metrics.requestTotal.WithLabelValues(method, endpoint, status).Inc()
}

func (h *Handler) recordError(method, endpoint, errType string) {
	h.metrics.errorTotal.WithLabelValues(method, endpoint, errType).Inc()
}

func (h *Handler) rateLimitMiddleware(c *gin.Context) {
	if !h.limiter.Allow() {
		log.Warn().
			Str("ip", c.ClientIP()).
			Str("path", c.Request.URL.Path).
			Msg("rate limit exceeded")
		c.JSON(http.StatusTooManyRequests, handler.NewErrorResponse("rate limit exceeded"))
		c.Abort()
		return
	}
	c.Next()
}

func (h *Handler) extractTraceContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	carrier := propagation.HeaderCarrier(c.Request.Header)
	return h.propagator.Extract(ctx, carrier)
}

func (h *Handler) CreateAppointment(c *gin.Context) {
	ctx := h.extractTraceContext(c)
	ctx, span := h.tracer.Start(ctx, "CreateAppointment",
		trace.WithAttributes(
			attribute.String("handler", "appointment"),
			attribute.String("operation", "create"),
		),
	)
	defer span.End()

	start := time.Now()
	method := "POST"
	endpoint := "/appointments"

	logger := log.With().
		Str("method", method).
		Str("endpoint", endpoint).
		Str("request_id", c.GetString("request_id")).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Logger()

	var req model.CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, "invalid request")
		h.recordError(method, endpoint, "validation")
		logger.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		h.recordMetrics(start, method, endpoint, "400")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.recordError(method, endpoint, "validation")
		logger.Warn().Err(err).Interface("request", req).Msg("validation failed")
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		h.recordMetrics(start, method, endpoint, "400")
		return
	}

	appointment := &model.Appointment{
		Base: model.Base{
			ID: uuid.New(),
		},
		ClinicID:    uuid.MustParse(req.ClinicID),
		ClinicianID: uuid.MustParse(req.ClinicianID),
		PatientID:   req.PatientID,
		ServiceID:   req.ServiceID,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      model.AppointmentStatusScheduled,
		Notes:       req.Notes,
	}

	err := h.service.CreateAppointment(ctx, appointment)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create appointment")
		if ctx.Err() == context.DeadlineExceeded {
			h.recordError(method, endpoint, "timeout")
			logger.Error().Err(err).Msg("request timeout")
			c.JSON(http.StatusGatewayTimeout, handler.NewErrorResponse(errRequestTimeout))
			h.recordMetrics(start, method, endpoint, "504")
			return
		}
		h.recordError(method, endpoint, "internal")
		logger.Error().Err(err).Interface("request", req).Msg("failed to create appointment")
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		h.recordMetrics(start, method, endpoint, "500")
		return
	}

	span.SetAttributes(
		attribute.String("appointment.id", appointment.ID.String()),
		attribute.String("clinic.id", appointment.ClinicID.String()),
	)

	logger.Info().
		Interface("appointment", appointment).
		Dur("duration", time.Since(start)).
		Msg("appointment created successfully")

	// Create outbox event
	payload, err := json.Marshal(appointment)
	if err != nil {
		log.Info().Msgf("failed to marshal appointment for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(ctx, &model.OutboxEvent{
			EventType: "APPOINTMENT_CREATE",
			Payload:   payload,
			Headers:   h.injectTraceContext(ctx),
		}); err != nil {
			log.Info().Msgf("failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusCreated, handler.NewSuccessResponse(appointment))
	h.recordMetrics(start, method, endpoint, "201")
}

func (h *Handler) injectTraceContext(ctx context.Context) map[string]string {
	headers := make(map[string]string)
	h.propagator.Inject(ctx, propagation.MapCarrier(headers))
	return headers
}

func (h *Handler) GetAppointment(c *gin.Context) {
	ctx := h.extractTraceContext(c)
	ctx, span := h.tracer.Start(ctx, "GetAppointment")
	defer span.End()

	start := time.Now()
	method := "GET"
	endpoint := "/appointments/:id"

	logger := log.With().
		Str("method", method).
		Str("endpoint", endpoint).
		Str("request_id", c.GetString("request_id")).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Logger()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetStatus(codes.Error, "invalid appointment ID")
		h.recordError(method, endpoint, "validation")
		logger.Warn().Err(err).Msg("invalid appointment ID")
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(errInvalidID))
		h.recordMetrics(start, method, endpoint, "400")
		return
	}

	span.SetAttributes(attribute.String("appointment.id", id.String()))

	// Try cache first
	cacheKey := fmt.Sprintf(cacheKeyAppointment, id.String())
	if cached, found := h.cache.Get(cacheKey); found {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		h.recordCacheMetrics(true, "appointment")
		logger.Info().Msg("appointment retrieved from cache")
		c.JSON(http.StatusOK, handler.NewSuccessResponse(cached))
		h.recordMetrics(start, method, endpoint, "200")
		return
	}
	span.SetAttributes(attribute.Bool("cache.hit", false))
	h.recordCacheMetrics(false, "appointment")

	result, err := h.breaker.Execute(func() (interface{}, error) {
		return h.service.GetAppointment(ctx, id)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			h.recordError(method, endpoint, "circuit_open")
			logger.Error().Msg("circuit breaker is open")
			c.JSON(http.StatusServiceUnavailable, handler.NewErrorResponse(errCircuitOpen))
			h.recordMetrics(start, method, endpoint, "503")
			return
		}
		h.recordError(method, endpoint, "internal")
		logger.Error().Err(err).Msg("failed to get appointment")
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		h.recordMetrics(start, method, endpoint, "500")
		return
	}

	appointment := result.(*model.Appointment)
	// Cache the result
	h.cache.Set(cacheKey, appointment, cache.DefaultExpiration)

	logger.Info().
		Interface("appointment", appointment).
		Dur("duration", time.Since(start)).
		Msg("appointment retrieved successfully")

	c.JSON(http.StatusOK, handler.NewSuccessResponse(appointment))
	h.recordMetrics(start, method, endpoint, "200")
}

func (h *Handler) ListAppointments(c *gin.Context) {
	start := time.Now()
	method := "GET"
	endpoint := "/appointments"

	logger := log.With().
		Str("method", method).
		Str("endpoint", endpoint).
		Str("request_id", c.GetString("request_id")).
		Logger()

	clinicID, err := uuid.Parse(c.Query("clinic_id"))
	if err != nil {
		h.recordError(method, endpoint, "validation")
		logger.Warn().Err(err).Msg("invalid clinic ID")
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(errInvalidClinicID))
		h.recordMetrics(start, method, endpoint, "400")
		return
	}

	filters := &model.AppointmentFilters{
		ClinicID:    clinicID,
		ClinicianID: uuid.Nil,
		PatientID:   uuid.Nil,
		Status:      "",
		StartDate:   time.Time{},
		EndDate:     time.Time{},
	}

	// Add optional filters with validation
	if id := c.Query("clinician_id"); id != "" {
		clinicianID, err := uuid.Parse(id)
		if err != nil {
			h.recordError(method, endpoint, "validation")
			logger.Warn().Err(err).Msg("invalid clinician ID")
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse(errInvalidClinicID))
			h.recordMetrics(start, method, endpoint, "400")
			return
		}
		filters.ClinicianID = clinicianID
	}

	if id := c.Query("patient_id"); id != "" {
		patientID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid patient ID"))
			return
		}
		filters.PatientID = patientID
	}

	if status := c.Query("status"); status != "" {
		filters.Status = model.AppointmentStatus(status)
	}

	if date := c.Query("start_date"); date != "" {
		startDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid start_date format"))
			return
		}
		filters.StartDate = startDate
	}

	if date := c.Query("end_date"); date != "" {
		endDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid end_date format"))
			return
		}
		filters.EndDate = endDate
	}

	result, err := h.breaker.Execute(func() (interface{}, error) {
		return h.service.ListAppointments(c.Request.Context(), filters)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			h.recordError(method, endpoint, "circuit_open")
			logger.Error().Msg("circuit breaker is open")
			c.JSON(http.StatusServiceUnavailable, handler.NewErrorResponse(errCircuitOpen))
			h.recordMetrics(start, method, endpoint, "503")
			return
		}
		h.recordError(method, endpoint, "internal")
		logger.Error().Err(err).Interface("filters", filters).Msg("failed to list appointments")
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		h.recordMetrics(start, method, endpoint, "500")
		return
	}

	appointments := result.([]*model.Appointment)
	logger.Info().
		Int("count", len(appointments)).
		Interface("filters", filters).
		Dur("duration", time.Since(start)).
		Msg("appointments retrieved successfully")

	c.JSON(http.StatusOK, handler.NewSuccessResponse(appointments))
	h.recordMetrics(start, method, endpoint, "200")
}

func (h *Handler) UpdateAppointment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultTimeout)
	defer cancel()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid appointment ID"))
		return
	}

	var req model.UpdateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(err.Error()))
		return
	}

	_, err = h.service.GetAppointment(ctx, id)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, handler.NewErrorResponse("request timeout"))
			return
		}
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	appointment := &model.Appointment{
		Base: model.Base{
			ID: id,
		},
		StartTime: *req.StartTime,
		EndTime:   *req.EndTime,
		Status:    *req.Status,
		Notes:     *req.Notes,
	}
	err = h.service.UpdateAppointment(ctx, appointment)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, handler.NewErrorResponse("request timeout"))
			return
		}
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Invalidate cache
	h.invalidateAppointmentCache(id)
	// Invalidate availability cache for the affected date
	if req.StartTime != nil {
		h.invalidateAvailabilityCache(appointment.ClinicianID, req.StartTime.Format("2006-01-02"))
	}

	// Create outbox event for update
	payload, err := json.Marshal(appointment)
	if err != nil {
		log.Info().Msgf("failed to marshal appointment for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(ctx, &model.OutboxEvent{
			EventType: "APPOINTMENT_UPDATE",
			Payload:   payload,
		}); err != nil {
			log.Info().Msgf("failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) DeleteAppointment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), defaultTimeout)
	defer cancel()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse("invalid appointment ID"))
		return
	}

	_, err = h.service.GetAppointment(ctx, id)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, handler.NewErrorResponse("request timeout"))
			return
		}
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	if err := h.service.DeleteAppointment(ctx, id); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, handler.NewErrorResponse("request timeout"))
			return
		}
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		return
	}

	// Invalidate cache
	h.invalidateAppointmentCache(id)
	// Get appointment details before deletion to invalidate availability cache
	if appointment, err := h.service.GetAppointment(ctx, id); err == nil {
		h.invalidateAvailabilityCache(appointment.ClinicianID, appointment.StartTime.Format("2006-01-02"))
	}

	// Create outbox event for delete
	payload, err := json.Marshal(map[string]interface{}{"id": id})
	if err != nil {
		log.Info().Msgf("failed to marshal appointment ID for event: %v", err)
	} else {
		if err := h.outboxRepo.Create(ctx, &model.OutboxEvent{
			EventType: "APPOINTMENT_DELETE",
			Payload:   payload,
		}); err != nil {
			log.Info().Msgf("failed to create outbox event: %v", err)
		}
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(nil))
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	appointments := r.Group("/appointments")
	appointments.Use(otelgin.Middleware("appointment-service"))
	appointments.Use(h.rateLimitMiddleware)
	{
		appointments.GET("/health", h.HealthCheck)
		appointments.GET("/availability", h.GetClinicianAvailability)
		appointments.POST("", h.CreateAppointment)
		appointments.GET("", h.ListAppointments)
		appointments.GET("/:id", h.GetAppointment)
		appointments.PUT("/:id", h.UpdateAppointment)
		appointments.DELETE("/:id", h.DeleteAppointment)
	}
}

func (h *Handler) GetClinicianAvailability(c *gin.Context) {
	start := time.Now()
	method := "GET"
	endpoint := "/appointments/availability"
	logger := log.With().
		Str("method", method).
		Str("endpoint", endpoint).
		Str("request_id", c.GetString("request_id")).
		Logger()

	clinicianID, err := uuid.Parse(c.Query("clinician_id"))
	if err != nil {
		h.recordError(method, endpoint, "validation")
		logger.Warn().Err(err).Msg("invalid clinician ID")
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(errInvalidClinicID))
		h.recordMetrics(start, method, endpoint, "400")
		return
	}

	dateStr := c.Query("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.recordError(method, endpoint, "validation")
		logger.Warn().Err(err).Msg("invalid date format")
		c.JSON(http.StatusBadRequest, handler.NewErrorResponse(errInvalidDate))
		h.recordMetrics(start, method, endpoint, "400")
		return
	}

	// Try cache first
	cacheKey := fmt.Sprintf(cacheKeyAvailability, clinicianID.String(), dateStr)
	if cached, found := h.cache.Get(cacheKey); found {
		h.recordCacheMetrics(true, "availability")
		logger.Info().Msg("availability retrieved from cache")
		c.JSON(http.StatusOK, handler.NewSuccessResponse(cached))
		h.recordMetrics(start, method, endpoint, "200")
		return
	}
	h.recordCacheMetrics(false, "availability")

	result, err := h.breaker.Execute(func() (interface{}, error) {
		return h.service.GetClinicianAvailability(c.Request.Context(), clinicianID, date)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			h.recordError(method, endpoint, "circuit_open")
			logger.Error().Msg("circuit breaker is open")
			c.JSON(http.StatusServiceUnavailable, handler.NewErrorResponse(errCircuitOpen))
			h.recordMetrics(start, method, endpoint, "503")
			return
		}
		h.recordError(method, endpoint, "internal")
		logger.Error().Err(err).Msg("failed to get availability")
		c.JSON(http.StatusInternalServerError, handler.NewErrorResponse(err.Error()))
		h.recordMetrics(start, method, endpoint, "500")
		return
	}

	slots := result.([]*model.TimeSlot)
	// Cache the result
	h.cache.Set(cacheKey, slots, cache.DefaultExpiration)

	logger.Info().
		Int("slot_count", len(slots)).
		Str("clinician_id", clinicianID.String()).
		Str("date", dateStr).
		Dur("duration", time.Since(start)).
		Msg("availability retrieved successfully")

	c.JSON(http.StatusOK, handler.NewSuccessResponse(slots))
	h.recordMetrics(start, method, endpoint, "200")
}

func (h *Handler) RegisterRoutesWithEvents(r *gin.RouterGroup, eventTracker *event.EventTrackerMiddleware) {
	appointments := r.Group("/appointments")
	{
		appointments.POST("", eventTracker.TrackEvent("appointment", "create"), h.CreateAppointment)
		appointments.PUT("/:id", eventTracker.TrackEvent("appointment", "update"), h.UpdateAppointment)
		appointments.DELETE("/:id", eventTracker.TrackEvent("appointment", "delete"), h.DeleteAppointment)
		appointments.GET("", h.ListAppointments)
		appointments.GET("/:id", h.GetAppointment)
	}
}

// HealthCheck returns the health status of the appointment service
func (h *Handler) HealthCheck(c *gin.Context) {
	status := "healthy"
	state := h.breaker.State()

	if state == gobreaker.StateOpen {
		status = "degraded"
	}

	health := map[string]interface{}{
		"status":          status,
		"circuit_breaker": state.String(),
		"cache": map[string]interface{}{
			"items": len(h.cache.Items()),
		},
	}

	c.JSON(http.StatusOK, handler.NewSuccessResponse(health))
}

func (h *Handler) recordCacheMetrics(hit bool, cacheType string) {
	if hit {
		h.metrics.cacheHits.WithLabelValues(cacheType).Inc()
	} else {
		h.metrics.cacheMisses.WithLabelValues(cacheType).Inc()
	}
}

func (h *Handler) invalidateAppointmentCache(id uuid.UUID) {
	cacheKey := fmt.Sprintf(cacheKeyAppointment, id.String())
	h.cache.Delete(cacheKey)
}

func (h *Handler) invalidateAvailabilityCache(clinicianID uuid.UUID, date string) {
	cacheKey := fmt.Sprintf(cacheKeyAvailability, clinicianID.String(), date)
	h.cache.Delete(cacheKey)
}

func (h *Handler) startOperation(ctx context.Context, name string) (context.Context, trace.Span) {
	return h.tracer.Start(ctx, name,
		trace.WithAttributes(
			attribute.String("component", "appointment-handler"),
			attribute.String("operation", name),
		),
	)
}

func (h *Handler) recordSpanError(span trace.Span, err error, msg string) {
	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
	span.SetAttributes(attribute.String("error", err.Error()))
}

func initTracer() (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(0.1), // Sample 10% of traces
		)),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("appointment-service"),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("environment", os.Getenv("APP_ENV")),
		)),
	}

	if exporterEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); exporterEndpoint != "" {
		exporter, err := otlptrace.New(
			context.Background(),
			otlptracegrpc.NewClient(
				otlptracegrpc.WithEndpoint(exporterEndpoint),
				otlptracegrpc.WithInsecure(),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return tp, nil
}
