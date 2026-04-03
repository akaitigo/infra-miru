package config_test

import (
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/config"
)

func setAllEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("PORT", "9090")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("KUBECONFIG", "/home/test/.kube/config")
	t.Setenv("PROMETHEUS_URL", "http://prometheus:9090")
}

func clearAllEnvVars(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT", "DB_HOST", "DB_PORT", "DB_USER",
		"DB_PASSWORD", "DB_NAME", "KUBECONFIG", "PROMETHEUS_URL",
	} {
		t.Setenv(key, "")
	}
}

func TestLoad_AllRequiredVarsSet(t *testing.T) {
	setAllEnvVars(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "localhost")
	}
	if cfg.DBUser != "testuser" {
		t.Errorf("DBUser = %q, want %q", cfg.DBUser, "testuser")
	}
	if cfg.DBPassword != "testpass" {
		t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "testpass")
	}
	if cfg.DBName != "testdb" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "testdb")
	}
	if cfg.KubeConfig != "/home/test/.kube/config" {
		t.Errorf("KubeConfig = %q, want %q", cfg.KubeConfig, "/home/test/.kube/config")
	}
	if cfg.PrometheusURL != "http://prometheus:9090" {
		t.Errorf("PrometheusURL = %q, want %q", cfg.PrometheusURL, "http://prometheus:9090")
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	clearAllEnvVars(t)
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("KUBECONFIG", "/home/test/.kube/config")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
}

func TestLoad_MissingRequiredVars(t *testing.T) {
	tests := []struct {
		name    string
		setVars map[string]string
	}{
		{
			name: "missing DB_HOST",
			setVars: map[string]string{
				"DB_USER": "u", "DB_PASSWORD": "p", "DB_NAME": "d", "KUBECONFIG": "/k",
			},
		},
		{
			name: "missing DB_USER",
			setVars: map[string]string{
				"DB_HOST": "h", "DB_PASSWORD": "p", "DB_NAME": "d", "KUBECONFIG": "/k",
			},
		},
		{
			name: "missing DB_PASSWORD",
			setVars: map[string]string{
				"DB_HOST": "h", "DB_USER": "u", "DB_NAME": "d", "KUBECONFIG": "/k",
			},
		},
		{
			name: "missing DB_NAME",
			setVars: map[string]string{
				"DB_HOST": "h", "DB_USER": "u", "DB_PASSWORD": "p", "KUBECONFIG": "/k",
			},
		},
		{
			name: "missing KUBECONFIG",
			setVars: map[string]string{
				"DB_HOST": "h", "DB_USER": "u", "DB_PASSWORD": "p", "DB_NAME": "d",
			},
		},
		{
			name: "multiple missing vars",
			setVars: map[string]string{
				"DB_PASSWORD": "p", "DB_NAME": "d",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAllEnvVars(t)
			for k, v := range tt.setVars {
				t.Setenv(k, v)
			}

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestDatabaseURL(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DBHost:     "db.example.com",
		DBPort:     "5433",
		DBUser:     "admin",
		DBPassword: "secret",
		DBName:     "infradb",
	}

	want := "postgres://admin:secret@db.example.com:5433/infradb?sslmode=disable"
	got := cfg.DatabaseURL()
	if got != want {
		t.Errorf("DatabaseURL() = %q, want %q", got, want)
	}
}
