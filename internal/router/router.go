package router

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"github.com/jwalitptl/admin-api/internal/handler"
	"github.com/jwalitptl/admin-api/internal/middleware"
	"github.com/jwalitptl/admin-api/pkg/event"
)

type Handler interface {
	RegisterRoutes(*gin.RouterGroup)
}

type EventHandler interface {
	Handler
	RegisterRoutesWithEvents(*gin.RouterGroup, *event.EventTrackerMiddleware)
}

type Router struct {
	engine            *gin.Engine
	auth              *middleware.AuthMiddleware
	accountH          EventHandler
	authH             Handler
	clinicH           EventHandler
	rbacH             EventHandler
	appointmentH      EventHandler
	patientHandler    EventHandler
	permissionHandler EventHandler
	h                 *handler.Handler
	eventTracker      *event.EventTrackerMiddleware
	userHandler       EventHandler
	regionValidation  *middleware.RegionValidationMiddleware
	metrics           *routerMetrics
}

type routerMetrics struct {
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec
}

type RouterConfig struct {
	RateLimit     rate.Limit
	RateBurst     int
	CORSConfig    middleware.CORSConfig
	MetricsPrefix string
}

func NewRouter(
	auth *middleware.AuthMiddleware,
	accountH EventHandler,
	authH Handler,
	clinicH EventHandler,
	userH EventHandler,
	rbacH EventHandler,
	appointmentH EventHandler,
	patientHandler EventHandler,
	permissionHandler EventHandler,
	h *handler.Handler,
	eventTracker *event.EventTrackerMiddleware,
	regionValidation *middleware.RegionValidationMiddleware,
	config RouterConfig,
) *Router {
	// Set production mode
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New() // Use New() instead of Default() for more control

	// Initialize metrics
	metrics := initRouterMetrics(config.MetricsPrefix)

	r := &Router{
		engine:            engine,
		auth:              auth,
		accountH:          accountH,
		authH:             authH,
		clinicH:           clinicH,
		rbacH:             rbacH,
		appointmentH:      appointmentH,
		patientHandler:    patientHandler,
		permissionHandler: permissionHandler,
		h:                 h,
		eventTracker:      eventTracker,
		userHandler:       userH,
		regionValidation:  regionValidation,
		metrics:           metrics,
	}

	// Add core middlewares
	engine.Use(
		gin.Recovery(),
		middleware.Logger(),
		middleware.ErrorHandler(),
		r.metricsMiddleware(),
		middleware.Timeout(middleware.TimeoutConfig{Duration: 30 * time.Second}),
		middleware.RequestID(),
	)

	// Add CORS with config
	engine.Use(middleware.CORS(config.CORSConfig))

	// Configure rate limiter
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		RPS:   float64(config.RateLimit),
		Burst: config.RateBurst,
	})
	engine.Use(rateLimiter.RateLimit())

	return r
}

func (r *Router) Setup() {
	api := r.engine.Group("/api/v1")

	// Add version header
	api.Use(func(c *gin.Context) {
		c.Header("X-API-Version", "1.0")
		c.Next()
	})

	// Health check endpoints
	r.setupHealthCheck(api)

	// Region validation
	api.Use(r.regionValidation.ValidateRegion())
	api.Use(r.regionValidation.ValidateRequirements())

	// Public routes
	r.setupPublicRoutes(api)

	// Protected routes
	protected := api.Group("")
	protected.Use(
		r.auth.Authenticate(),
		r.auth.ValidatePermissions(),
	)
	r.setupProtectedRoutes(protected)
}

func (r *Router) setupHealthCheck(rg *gin.RouterGroup) {
	health := rg.Group("/health")
	{
		health.GET("/live", r.h.LivenessCheck)
		health.GET("/ready", r.h.ReadinessCheck)
		health.GET("/metrics", r.h.MetricsHandler)
	}
}

func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	r.authH.RegisterRoutes(rg)
	r.accountH.RegisterRoutesWithEvents(rg, r.eventTracker)
}

func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	// Patient routes with region-specific features
	patients := rg.Group("/patients")
	r.setupPatientRoutes(patients)

	// Register other protected routes
	r.userHandler.RegisterRoutesWithEvents(rg, r.eventTracker)
	r.clinicH.RegisterRoutesWithEvents(rg, r.eventTracker)
	r.rbacH.RegisterRoutesWithEvents(rg, r.eventTracker)
	r.appointmentH.RegisterRoutesWithEvents(rg, r.eventTracker)
	r.permissionHandler.RegisterRoutesWithEvents(rg, r.eventTracker)
}

func (r *Router) setupPatientRoutes(rg *gin.RouterGroup) {
	// Base patient routes
	r.patientHandler.RegisterRoutesWithEvents(rg, r.eventTracker)

	// Advanced features
	advanced := rg.Group("/advanced")
	advanced.Use(r.regionValidation.ValidateFeature("advanced_patient_profile"))
	r.setupAdvancedPatientRoutes(advanced)

	// HIPAA compliant routes
	hipaa := rg.Group("/records")
	hipaa.Use(r.regionValidation.ValidateFeature("hipaa_compliance"))
	r.setupHIPAARoutes(hipaa)
}

func (r *Router) setupAdvancedPatientRoutes(rg *gin.RouterGroup) {
	if h, ok := r.patientHandler.(AdvancedPatientHandler); ok {
		rg.POST("/bulk", h.BulkCreate)
		rg.POST("/import", h.ImportPatients)
	}
}

func (r *Router) setupHIPAARoutes(rg *gin.RouterGroup) {
	if h, ok := r.patientHandler.(HIPAACompliantHandler); ok {
		rg.POST("", h.AddMedicalRecord)
		rg.GET("", h.ListMedicalRecords)
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// Additional interfaces for feature-specific handlers
type AdvancedPatientHandler interface {
	BulkCreate(*gin.Context)
	ImportPatients(*gin.Context)
}

type HIPAACompliantHandler interface {
	AddMedicalRecord(*gin.Context)
	ListMedicalRecords(*gin.Context)
}

// Metrics initialization and middleware
func initRouterMetrics(prefix string) *routerMetrics {
	return &routerMetrics{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: prefix + "_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds",
			},
			[]string{"method", "path", "status"},
		),
		requestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: prefix + "_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		errorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: prefix + "_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "path", "type"},
		),
	}
}

func (r *Router) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		c.Next()

		status := fmt.Sprintf("%d", c.Writer.Status())
		duration := time.Since(start).Seconds()

		r.metrics.requestDuration.WithLabelValues(c.Request.Method, path, status).Observe(duration)
		r.metrics.requestTotal.WithLabelValues(c.Request.Method, path, status).Inc()

		if c.Writer.Status() >= 400 {
			r.metrics.errorTotal.WithLabelValues(c.Request.Method, path, "http").Inc()
		}
	}
}

func (r *Router) Use(middleware ...gin.HandlerFunc) {
	r.engine.Use(middleware...)
}

func (r *Router) GET(path string, handlers ...gin.HandlerFunc) {
	r.engine.GET(path, handlers...)
}
