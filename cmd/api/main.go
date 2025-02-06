package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jwalitptl/pkg/config"
	"github.com/jwalitptl/pkg/event"
	"github.com/jwalitptl/pkg/messaging/redis"
	"github.com/jwalitptl/pkg/worker"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/handler/account"
	"github.com/jwalitptl/admin-api/internal/handler/appointment"
	"github.com/jwalitptl/admin-api/internal/handler/auth"
	"github.com/jwalitptl/admin-api/internal/handler/clinic"
	"github.com/jwalitptl/admin-api/internal/handler/clinician"
	"github.com/jwalitptl/admin-api/internal/handler/patient"
	permissionHandler "github.com/jwalitptl/admin-api/internal/handler/permission"
	"github.com/jwalitptl/admin-api/internal/handler/rbac"
	"github.com/jwalitptl/admin-api/internal/middleware"
	"github.com/jwalitptl/admin-api/internal/repository/postgres"
	"github.com/jwalitptl/admin-api/internal/router"
	accountService "github.com/jwalitptl/admin-api/internal/service/account"
	appointmentService "github.com/jwalitptl/admin-api/internal/service/appointment"
	authService "github.com/jwalitptl/admin-api/internal/service/auth"
	clinicService "github.com/jwalitptl/admin-api/internal/service/clinic"
	clinicianService "github.com/jwalitptl/admin-api/internal/service/clinician"
	eventService "github.com/jwalitptl/admin-api/internal/service/event"
	patientService "github.com/jwalitptl/admin-api/internal/service/patient"
	permissionService "github.com/jwalitptl/admin-api/internal/service/permission"
	rbacService "github.com/jwalitptl/admin-api/internal/service/rbac"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Initialize database
	db, err := postgres.NewDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	accountRepo := postgres.NewAccountRepository(db)
	organizationRepo := postgres.NewOrganizationRepository(db)
	clinicRepo := postgres.NewClinicRepository(db)
	clinicianRepo := postgres.NewClinicianRepository(db)
	rbacRepo := postgres.NewRBACRepository(db)
	appointmentRepo := postgres.NewAppointmentRepository(db)
	patientRepo := postgres.NewPatientRepository(db)
	permRepo := postgres.NewPermissionRepository(db)

	// Initialize services
	accountSvc := accountService.NewService(accountRepo, organizationRepo)
	clinicSvc := clinicService.NewService(clinicRepo)
	clinicianSvc := clinicianService.NewService(clinicianRepo)
	rbacSvc := rbacService.NewService(rbacRepo)
	authSvc := authService.NewService(clinicianRepo, cfg.JWT)
	appointmentSvc := appointmentService.NewService(appointmentRepo)
	patientSvc := patientService.NewService(patientRepo)
	permService := permissionService.NewService(permRepo)

	// Initialize outbox repository
	outboxRepo := postgres.NewOutboxRepository(db)

	// Initialize event service with outbox repository
	eventSvc := eventService.NewService(outboxRepo)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(rbacSvc, authSvc)

	// Initialize event tracking middleware
	eventTracker := event.NewEventTrackerMiddleware(
		&cfg.EventTracking,
		eventSvc,
	)

	// Initialize handlers
	h := handler.NewHandler()
	accountHandler := account.NewHandler(accountSvc)
	authHandler := auth.NewHandler(authSvc)
	clinicHandler := clinic.NewHandler(clinicSvc)
	clinicianHandler := clinician.NewHandler(clinicianSvc, db)
	rbacHandler := rbac.NewHandler(rbacSvc)
	appointmentHandler := appointment.NewHandler(appointmentSvc)
	patientHandler := patient.NewHandler(patientSvc)
	permHandler := permissionHandler.NewHandler(permService)

	// Setup router
	r := router.NewRouter(
		authMiddleware,
		accountHandler,
		authHandler,
		clinicHandler,
		clinicianHandler,
		rbacHandler,
		appointmentHandler,
		patientHandler,
		permHandler,
		h,
	)

	// Register routes after router creation
	r.Setup()
	patientHandler.RegisterRoutesWithEvents(r.Engine().Group("/api/v1"), eventTracker)
	appointmentHandler.RegisterRoutesWithEvents(r.Engine().Group("/api/v1"), eventTracker)
	clinicHandler.RegisterRoutesWithEvents(r.Engine().Group("/api/v1"), eventTracker)
	clinicianHandler.RegisterRoutesWithEvents(r.Engine().Group("/api/v1"), eventTracker)

	// Create server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r.Engine(),
	}

	// Initialize Redis message broker
	broker, err := redis.NewRedisBroker(cfg.Redis.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	// Initialize and start outbox processor with broker
	outboxProcessor := worker.NewOutboxProcessor(outboxRepo, broker)
	go outboxProcessor.Start(context.Background())

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
}
