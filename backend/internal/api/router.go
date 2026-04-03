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

// NewRouter creates and configures a chi router with middleware and routes.
// Pass nil for deps to register only the health endpoint (useful for tests).
func NewRouter(deps *RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Timeout(requestTimeout))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", HealthHandler())

		if deps != nil {
			r.Get("/resources", ResourceHandler(deps.PodLister, deps.Analyzer))
			r.Get("/recommendations", RecommendationHandler(deps.PodLister, deps.Analyzer, deps.Calculator))
		}
	})

	return r
}
