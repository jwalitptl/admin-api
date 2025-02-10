package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/jwalitptl/admin-api/internal/config"
	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/handler/account"
	"github.com/jwalitptl/admin-api/internal/handler/appointment"
	authHandler "github.com/jwalitptl/admin-api/internal/handler/auth"
	"github.com/jwalitptl/admin-api/internal/handler/clinic"
	"github.com/jwalitptl/admin-api/internal/handler/health"
	"github.com/jwalitptl/admin-api/internal/handler/patient"
	permissionHandler "github.com/jwalitptl/admin-api/internal/handler/permission"
	"github.com/jwalitptl/admin-api/internal/handler/prometheus"
	rbacHandler "github.com/jwalitptl/admin-api/internal/handler/rbac"
	"github.com/jwalitptl/admin-api/internal/handler/user"
	"github.com/jwalitptl/admin-api/internal/middleware"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/internal/router"
	accountService "github.com/jwalitptl/admin-api/internal/service/account"
	appointmentService "github.com/jwalitptl/admin-api/internal/service/appointment"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/internal/service/auth"
	clinicService "github.com/jwalitptl/admin-api/internal/service/clinic"
	"github.com/jwalitptl/admin-api/internal/service/email"
	"github.com/jwalitptl/admin-api/internal/service/event"
	"github.com/jwalitptl/admin-api/internal/service/notification"
	patientService "github.com/jwalitptl/admin-api/internal/service/patient"
	permissionService "github.com/jwalitptl/admin-api/internal/service/permission"
	rbacService "github.com/jwalitptl/admin-api/internal/service/rbac"
	"github.com/jwalitptl/admin-api/internal/service/region"
	userService "github.com/jwalitptl/admin-api/internal/service/user"
	"github.com/jwalitptl/admin-api/pkg/event"
	"github.com/jwalitptl/admin-api/pkg/logger"
	"github.com/jwalitptl/admin-api/pkg/messaging/redis"
	"github.com/jwalitptl/admin-api/pkg/metrics"
	"github.com/jwalitptl/admin-api/pkg/worker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Initialize logger first
	l := logger.NewZerologAdapter(log.Logger)
	log.Logger = l.GetZerolog() // Set the global logger

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to load configuration")
	}

	// After loading config
	l.Debug().Interface("event_tracking", cfg.EventTracking).Msg("loaded config")

	// Initialize database
	db, err := sqlx.Connect("postgres", cfg.Database.URL)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories first
	baseRepo := postgres.NewBaseRepository(db)
	accountRepo := postgres.NewAccountRepository(baseRepo)
	organizationRepo := postgres.NewOrganizationRepository(baseRepo)
	clinicRepo := postgres.NewClinicRepository(baseRepo)
	userRepo := postgres.NewUserRepository(baseRepo)
	rbacRepo := postgres.NewRBACRepository(baseRepo)
	appointmentRepo := postgres.NewAppointmentRepository(baseRepo)
	patientRepo := postgres.NewPatientRepository(baseRepo)
	permRepo := postgres.NewPermissionRepository(baseRepo)
	outboxRepo := postgres.NewOutboxRepository(baseRepo)
	auditRepo := postgres.NewAuditRepository(baseRepo)
	regionRepo := postgres.NewRegionRepository(baseRepo)
	tokenRepo := postgres.NewTokenRepository(baseRepo)
	notificationRepo := postgres.NewNotificationRepository(baseRepo)

	// Initialize core services first
	emailSvc := email.NewService(cfg.Email)
	auditSvc := audit.NewService(auditRepo)
	jwtSvc := auth.NewJWTService(cfg.JWT.Secret)

	// Initialize Redis broker
	redisConfig := redis.Config{
		URL: "redis://redis:6379/0",
	}
	broker, err := redis.NewRedisBroker(redisConfig, l)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	// Initialize business services
	accountSvc := accountService.NewService(accountRepo, organizationRepo, emailSvc, auditSvc)
	clinicSvc := clinicService.NewService(clinicRepo, auditSvc)
	userSvc := userService.NewService(userRepo, emailSvc, tokenRepo, auditSvc)
	rbacSvc := rbacService.NewService(rbacRepo, auditSvc)
	authSvc := auth.NewService(userRepo, emailSvc, jwtSvc, tokenRepo, cfg.JWT)
	notificationSvc := notification.NewService(
		notificationRepo,
		emailSvc,
		broker,
		auditSvc,
	)
	appointmentSvc := appointmentService.NewService(
		appointmentRepo,
		notificationSvc,
		clinicianRepo,
		auditSvc,
	)
	permSvc := permissionService.NewService(permRepo, auditSvc)
	eventSvc := event.NewService(outboxRepo, broker, auditSvc)
	patientSvc := patientService.NewService(patientRepo)
	regionSvc := region.NewService(regionRepo, geoIP, defaultConfig)

	// Initialize handlers with correct dependencies
	h := handler.NewHandler()
	accountHandler := account.NewHandler(accountSvc)
	authHandler := authHandler.NewHandler(authSvc)
	clinicHandler := clinic.NewHandler(clinicSvc, outboxRepo)
	userHandler := user.NewHandler(userSvc)
	rbacHandler := rbacHandler.NewHandler(rbacSvc)
	appointmentHandler := appointment.NewHandler(appointmentSvc, outboxRepo)
	permHandler := permissionHandler.NewHandler(permSvc, outboxRepo)
	patientHandler := patient.NewHandler(patientSvc, outboxRepo)
	auditHandler := audit.NewHandler(auditSvc)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(rbacSvc, authSvc)
	hipaaMiddleware := middleware.NewHIPAAMiddleware(auditSvc)

	// Initialize event tracking middleware
	eventTracker := event.NewEventTrackerMiddleware(eventSvc)

	// Initialize region middleware
	regionMiddleware := middleware.NewRegionMiddleware(regionSvc)
	regionValidation := middleware.NewRegionValidationMiddleware(defaultConfig)

	// Setup router
	r := router.NewRouter(
		authMiddleware,
		hipaaMiddleware,
		regionMiddleware,
		regionValidation,
		accountHandler,
		authHandler,
		clinicHandler,
		userHandler,
		rbacHandler,
		appointmentHandler,
		permHandler,
		patientHandler,
		h,
		eventTracker,
	)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		RPS:   cfg.RateLimit.RequestsPerSecond,
		Burst: cfg.RateLimit.Burst,
	})

	// Add middlewares
	r.Use(middleware.Logger())
	r.Use(middleware.ErrorHandler())
	if cfg.RateLimit.Enabled {
		r.Use(rateLimiter.RateLimit())
	}

	// Add health check routes
	healthHandler := health.NewHandler(db)
	healthHandler.RegisterRoutes(r.Engine().Group("/"))

	// Add metrics if enabled
	if cfg.Monitoring.PrometheusEnabled {
		p := prometheus.New()
		r.Use(p.Middleware())
		r.GET(cfg.Monitoring.MetricsPath, p.Handler())
	}

	// Register routes after router creation
	r.Setup()

	// Initialize and start outbox processor with broker
	outboxConfig := worker.OutboxProcessorConfig{
		BatchSize:     100,
		PollInterval:  time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Minute,
	}
	outboxProcessor := worker.NewOutboxProcessor(
		outboxRepo,
		broker,
		outboxConfig,
		l,
		metrics.New("outbox_processor"),
	)
	processorCtx, processorCancel := context.WithCancel(context.Background())
	defer processorCancel()
	go outboxProcessor.Start(processorCtx)

	// Initialize audit cleanup worker
	auditCleanup := worker.NewAuditCleanupWorker(
		auditRepo,
		cfg.Audit.RetentionDays,
		24*time.Hour, // Run cleanup daily
	)

	// Start audit cleanup worker
	go auditCleanup.Start(processorCtx)

	// Register audit routes
	r.Engine().Group("/audit").Use(authMiddleware.Authenticate()).
		Use(authMiddleware.RequireRole(model.UserTypeAdmin)).
		GET("/logs", auditHandler.ListLogs)

	// Create server
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        r.Engine(),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	l.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		l.Fatal().Err(err).Msg("server forced to shutdown")
	}

	l.Info().Msg("server exited properly")

	// Initialize tracer
	tp, err := initTracer()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to initialize tracer")
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			l.Error().Err(err).Msg("failed to shutdown tracer provider")
		}
	}()
}

func initTracer() (*trace.TracerProvider, error) {
	// Initialize OpenTelemetry tracer
	exporter, err := otlptrace.New(context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint("otel-collector:4317"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
