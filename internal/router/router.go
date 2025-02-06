package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/middleware"
)

type Router struct {
	engine            *gin.Engine
	auth              *middleware.AuthMiddleware
	accountH          Handler
	authH             Handler
	clinicH           Handler
	clinicianH        Handler
	rbacH             Handler
	appointmentH      Handler
	patientHandler    Handler
	permissionHandler Handler
	h                 *handler.Handler
}

type Handler interface {
	RegisterRoutes(*gin.RouterGroup)
}

func NewRouter(
	auth *middleware.AuthMiddleware,
	accountH Handler,
	authH Handler,
	clinicH Handler,
	clinicianH Handler,
	rbacH Handler,
	appointmentH Handler,
	patientHandler Handler,
	permissionHandler Handler,
	h *handler.Handler,
) *Router {
	engine := gin.Default()

	// Add middlewares
	engine.Use(gin.Recovery())
	engine.Use(middleware.Logger())

	// Rate limit: 100 requests per minute
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)
	engine.Use(rateLimiter.RateLimit())

	// Add health check endpoint
	engine.GET("/api/v1/health", h.HealthCheck)

	return &Router{
		engine:            engine,
		auth:              auth,
		accountH:          accountH,
		authH:             authH,
		clinicH:           clinicH,
		clinicianH:        clinicianH,
		rbacH:             rbacH,
		appointmentH:      appointmentH,
		patientHandler:    patientHandler,
		permissionHandler: permissionHandler,
		h:                 h,
	}
}

func (r *Router) Setup() {
	// Public routes
	api := r.engine.Group("/api/v1")
	{
		// Auth routes (no auth required)
		r.authH.RegisterRoutes(api)

		// Public endpoints (no auth required)
		r.accountH.RegisterRoutes(api)
		r.clinicianH.RegisterRoutes(api)
		r.patientHandler.RegisterRoutes(api)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(r.auth.Authenticate())
	{
		r.clinicH.RegisterRoutes(protected)
		r.rbacH.RegisterRoutes(protected)
		r.appointmentH.RegisterRoutes(protected)
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
