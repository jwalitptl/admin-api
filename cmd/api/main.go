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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/jwalitptl/admin-api/internal/config"
	"github.com/jwalitptl/admin-api/internal/handler/health"
	"github.com/jwalitptl/admin-api/internal/handler/prometheus"
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
	"github.com/jwalitptl/admin-api/internal/service/geoip"
	"github.com/jwalitptl/admin-api/internal/service/notification"
	patientService "github.com/jwalitptl/admin-api/internal/service/patient"
	permissionService "github.com/jwalitptl/admin-api/internal/service/permission"
	rbacService "github.com/jwalitptl/admin-api/internal/service/rbac"
	"github.com/jwalitptl/admin-api/internal/service/region"
	userService "github.com/jwalitptl/admin-api/internal/service/user"
	pkg_event "github.com/jwalitptl/admin-api/pkg/event"
	"github.com/jwalitptl/admin-api/pkg/messaging"
	"github.com/jwalitptl/admin-api/pkg/messaging/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/jwalitptl/admin-api/internal/handler/account"
	"github.com/jwalitptl/admin-api/internal/handler/appointment"
	authHandler "github.com/jwalitptl/admin-api/internal/handler/auth"
	"github.com/jwalitptl/admin-api/internal/handler/clinic"
	"github.com/jwalitptl/admin-api/internal/handler/organization"
	patientHandler "github.com/jwalitptl/admin-api/internal/handler/patient"
	permissionHandler "github.com/jwalitptl/admin-api/internal/handler/permission"
	rbacHandler "github.com/jwalitptl/admin-api/internal/handler/rbac"
	"github.com/jwalitptl/admin-api/internal/handler/user"
	pkg_auth "github.com/jwalitptl/admin-api/pkg/auth"
)

func main() {
	// Set config file path
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("internal/config") // Look for config in internal/config directory

	// Initialize logger
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	var cfg config.Config
	if err := config.Load(&cfg); err != nil {
		logger.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Initialize database
	dbURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost", // Change to localhost since we're using port forwarding
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	baseRepo := postgres.NewBaseRepository(db)
	accountRepo := postgres.NewAccountRepository(baseRepo)
	organizationRepo := postgres.NewOrganizationRepository(baseRepo)
	clinicRepo := postgres.NewClinicRepository(baseRepo)
	clinicianRepo := postgres.NewClinicianRepository(baseRepo)
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
	medicalRecordRepo := postgres.NewMedicalRecordRepository(baseRepo)

	// Initialize core services
	emailSvc := email.NewService(cfg.Email)
	auditSvc := audit.NewService(auditRepo)
	jwtSvc := pkg_auth.NewJWTService(cfg.JWT.Secret)
	geoIP := geoip.NewService(cfg.GeoIP)
	defaultConfig := &model.RegionConfig{}

	// Initialize broker
	redisConfig := redis.Config{
		URL: cfg.Redis.URL,
	}
	broker, err := redis.NewRedisBroker(redisConfig, &logger)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	// Initialize event service
	eventSvc := pkg_event.NewService(outboxRepo, messaging.NewBrokerAdapter(broker), auditSvc)

	// Initialize business services
	accountSvc := accountService.NewService(accountRepo, organizationRepo, emailSvc, auditSvc)
	clinicSvc := clinicService.NewService(clinicRepo, auditSvc)
	patientSvc := patientService.NewService(patientRepo, medicalRecordRepo, appointmentRepo, auditSvc)
	userSvc := userService.NewService(userRepo, emailSvc, tokenRepo, auditSvc, patientSvc)
	rbacSvc := rbacService.NewService(rbacRepo, auditSvc)
	authSvc := auth.NewService(userRepo, jwtSvc, tokenRepo, emailSvc, auditSvc)
	notificationSvc := notification.NewService(notificationRepo, emailSvc, broker, auditSvc)
	appointmentSvc := appointmentService.NewService(appointmentRepo, notificationSvc, clinicianRepo, auditSvc, outboxRepo, log.Logger)
	permSvc := permissionService.NewService(permRepo, auditSvc)
	regionSvc := region.NewService(regionRepo, geoIP, auditSvc, defaultConfig)

	// Initialize event tracking middleware
	eventTracker := pkg_event.NewEventTrackerMiddleware(eventSvc)

	repos := &router.Repositories{
		RBAC:  rbacRepo,
		Audit: auditRepo,
	}

	// Initialize router
	r := router.NewRouter(router.Config{
		AuthMiddleware:      middleware.NewAuthMiddleware(rbacSvc, authSvc),
		RegionMiddleware:    middleware.NewRegionMiddleware(regionSvc, middleware.RegionConfig{RequireRegion: true}),
		RegionValidation:    middleware.NewRegionValidationMiddleware(defaultConfig),
		AccountHandler:      account.NewHandler(accountSvc),
		OrganizationHandler: organization.NewHandler(accountSvc),
		AuthHandler:         authHandler.NewHandler(authSvc),
		ClinicHandler:       clinic.NewHandler(clinicSvc, outboxRepo),
		UserHandler:         user.NewHandler(userSvc, patientSvc, db),
		RBACHandler:         rbacHandler.NewHandler(rbacSvc, outboxRepo),
		AppointmentHandler:  appointment.NewHandler(appointmentSvc, outboxRepo, userRepo),
		PermissionHandler:   permissionHandler.NewHandler(permSvc, outboxRepo),
		PatientHandler:      patientHandler.NewHandler(patientSvc, outboxRepo, regionSvc),
		EventTracker:        eventTracker,
	}, repos)

	// Setup routes
	r.Setup()

	// Register health check routes
	healthHandler := health.NewHandler(db)
	healthHandler.RegisterRoutes(r.Engine().Group(""))

	// Add middleware
	r.Engine().Use(middleware.Logger())
	r.Engine().Use(middleware.ErrorHandler())

	// Add rate limiter
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		RPS:   cfg.RateLimit.RequestsPerSecond,
		Burst: cfg.RateLimit.Burst,
	})
	if cfg.RateLimit.Enabled {
		r.Engine().Use(rateLimiter.RateLimit())
	}

	// Add metrics
	if cfg.Monitoring.PrometheusEnabled {
		p := prometheus.New()
		r.Use(p.Middleware())
		r.GET(cfg.Monitoring.MetricsPath, p.Handler())
	}

	// Start server
	srv := &http.Server{
		Addr:    ":8081", // Use a different port temporarily
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	// Initialize tracer
	tp, err := initTracer()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize tracer")
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("failed to shutdown tracer provider")
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
