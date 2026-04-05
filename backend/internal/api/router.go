package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
)

const requestTimeout = 60 * time.Second

// RouterDeps holds optional dependencies for registering resource-related routes.
// When nil, only the health endpoint is registered.
type RouterDeps struct {
	PodLister  PodLister
	Analyzer   *analyzer.Analyzer
	Calculator *cost.Calculator
}

// RouterConfig holds configuration for the router middleware.
type RouterConfig struct {
	JWTSecret   string
	CORSOrigins []string
}

// NewRouter creates and configures a chi router with middleware and routes.
// Pass nil for deps to register only the health endpoint (useful for tests).
// Pass nil for cfg to use development defaults (no auth, localhost CORS).
func NewRouter(deps *RouterDeps, cfg *RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	corsOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	if cfg != nil && len(cfg.CORSOrigins) > 0 {
		corsOrigins = cfg.CORSOrigins
	}

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Timeout(requestTimeout))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health endpoint is public (no auth required).
		r.Get("/health", HealthHandler())

		// Protected routes require JWT authentication.
		r.Group(func(r chi.Router) {
			if cfg != nil && cfg.JWTSecret != "" {
				r.Use(JWTAuth(cfg.JWTSecret))
			}

			if deps != nil {
				r.Get("/resources", ResourceHandler(deps.PodLister, deps.Analyzer))
				r.Get("/recommendations", RecommendationHandler(deps.PodLister, deps.Analyzer, deps.Calculator))
				r.Get("/schedules", ScheduleHandler(deps.PodLister, deps.Analyzer))
				r.Get("/cronhpa/{deployment}", CronHPAHandler(deps.PodLister, deps.Analyzer))
			}
		})
	})

	return r
}
