// Package main is the entrypoint for the Infra-Miru API server.
package main

import (
	"context"
	"log"

	"github.com/akaitigo/infra-miru/backend/internal/api"
	"github.com/akaitigo/infra-miru/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	router := api.NewRouter(nil)
	srv := api.NewServer(cfg.Port, router)

	log.Printf("infra-miru server starting on port %s", cfg.Port)

	if err := srv.Run(context.Background()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
