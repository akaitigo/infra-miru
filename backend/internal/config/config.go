// Package config handles application configuration loading and validation
// from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration values loaded from environment variables.
type Config struct {
	Port          string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	KubeConfig    string
	PrometheusURL string
}

// DatabaseURL returns a PostgreSQL connection string built from the config fields.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

// Load reads configuration from environment variables. Required variables must be
// set; otherwise an error describing all missing variables is returned.
func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnvOrDefault("PORT", "8080"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        getEnvOrDefault("DB_PORT", "5432"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		KubeConfig:    os.Getenv("KUBECONFIG"),
		PrometheusURL: os.Getenv("PROMETHEUS_URL"),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	type requiredField struct {
		name  string
		value string
	}

	fields := []requiredField{
		{name: "DB_HOST", value: c.DBHost},
		{name: "DB_USER", value: c.DBUser},
		{name: "DB_PASSWORD", value: c.DBPassword},
		{name: "DB_NAME", value: c.DBName},
		{name: "KUBECONFIG", value: c.KubeConfig},
	}

	var missing []string
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			missing = append(missing, f.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"required environment variables not set: %w",
			errors.New(strings.Join(missing, ", ")),
		)
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
