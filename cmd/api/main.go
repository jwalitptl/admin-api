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
	accountService "github.com/jwalitptl/admin-api/internal/service/account"
	appointmentService "github.com/jwalitptl/admin-api/internal/service/appointment"
	clinicService "github.com/jwalitptl/admin-api/internal/service/clinic"
	eventService "github.com/jwalitptl/admin-api/internal/service/event"
	patientService "github.com/jwalitptl/admin-api/internal/service/patient"
	permissionService "github.com/jwalitptl/admin-api/internal/service/permission"
	rbacService "github.com/jwalitptl/admin-api/internal/service/rbac"
	"github.com/jwalitptl/admin-api/internal/service/region"
	userService "github.com/jwalitptl/admin-api/internal/service/user"
	"github.com/jwalitptl/admin-api/pkg/event"
	"github.com/jwalitptl/admin-api/pkg/messaging/redis"
	"github.com/jwalitptl/admin-api/pkg/worker"
	"golang.org/x/time/rate"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// After loading config
	fmt.Printf("DEBUG: Loaded config: %+v\n", cfg.EventTracking)

	// Initialize database
	db, err := sqlx.Connect("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	accountRepo := postgres.NewAccountRepository(db)
	organizationRepo := postgres.NewOrganizationRepository(db)
	clinicRepo := postgres.NewClinicRepository(db)
	userRepo := postgres.NewUserRepository(db)
	rbacRepo := postgres.NewRBACRepository(db)
	appointmentRepo := postgres.NewAppointmentRepository(db)
	patientRepo := postgres.NewPatientRepository(db)
	permRepo := postgres.NewPermissionRepository(db)
	outboxRepo := postgres.NewOutboxRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	regionRepo := postgres.NewRegionRepository(db)
	tokenRepo := postgres.NewTokenRepository(db)

	// Initialize Redis message broker
	broker, err := redis.NewRedisBroker("redis://redis:6379/0")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	// Initialize services
	accountSvc := accountService.NewService(accountRepo, organizationRepo, userRepo)
	clinicSvc := clinicService.NewService(clinicRepo)
	userSvc := userService.NewService(userRepo, organizationRepo)
	rbacSvc := rbacService.NewService(rbacRepo)
	jwtService := auth.NewJWTService(cfg.JWT.Secret)
	authSvc := auth.NewService(
		userRepo,
		tokenRepo,
		emailSvc,
		jwtService,
		cfg.JWT,
	)
	appointmentSvc := appointmentService.NewService(appointmentRepo)
	permService := permissionService.NewService(permRepo)
	eventSvc := eventService.NewService(outboxRepo, broker)
	patientSvc := patientService.NewService(patientRepo)
	auditSvc := audit.NewService(auditRepo)
	regionSvc := region.NewService(regionRepo, geoIP, defaultConfig)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(rbacSvc, authSvc)
	hipaaMiddleware := middleware.NewHIPAAMiddleware(auditSvc)

	// Initialize event tracking middleware
	eventTracker := event.NewEventTrackerMiddleware(eventSvc)

	// Initialize region middleware
	regionMiddleware := middleware.NewRegionMiddleware(regionSvc)
	regionValidation := middleware.NewRegionValidationMiddleware(defaultConfig)

	// Initialize handlers
	h := handler.NewHandler()
	accountHandler := account.NewHandler(accountSvc)
	authHandler := authHandler.NewHandler(authSvc)
	clinicHandler := clinic.NewHandler(clinicSvc, outboxRepo)
	userHandler := user.NewHandler(userSvc, db)
	rbacHandler := rbacHandler.NewHandler(rbacSvc)
	appointmentHandler := appointment.NewHandler(appointmentSvc, outboxRepo)
	permHandler := permissionHandler.NewHandler(permService, outboxRepo)
	patientHandler := patient.NewHandler(patientSvc, outboxRepo)
	auditHandler := audit.NewHandler(auditSvc)

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
	rateLimiter := middleware.NewRateLimiter(rate.Limit(cfg.RateLimit.RequestsPerSecond), cfg.RateLimit.Burst)

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
	outboxProcessor := worker.NewOutboxProcessor(outboxRepo, broker)
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
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited properly")

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
