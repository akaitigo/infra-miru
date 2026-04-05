// Package main is the entrypoint for the Infra-Miru API server.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/api"
	"github.com/akaitigo/infra-miru/backend/internal/config"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
	"github.com/akaitigo/infra-miru/backend/internal/db"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Database connection and migration.
	dbURL := cfg.DatabaseURL()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	log.Println("database connected")

	err = db.RunMigrations(dbURL)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Println("database migrations applied")

	// Kubernetes client initialization.
	k8sClient, err := k8s.Connect(cfg.KubeConfig)
	if err != nil {
		return fmt.Errorf("connect to kubernetes: %w", err)
	}

	log.Println("kubernetes client connected")

	// Build dependencies for the router.
	deps := &api.RouterDeps{
		PodLister:  k8sClient,
		Analyzer:   analyzer.NewAnalyzer(),
		Calculator: cost.NewCalculator(),
	}

	routerCfg := &api.RouterConfig{
		CORSOrigins: cfg.CORSOrigins,
		JWTSecret:   cfg.JWTSecret,
	}

	router := api.NewRouter(deps, routerCfg)
	srv := api.NewServer(cfg.Port, router)

	log.Printf("infra-miru server starting on port %s", cfg.Port)

	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server: %w", err)
	}

	return nil
}
