package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/api"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		wantStatus     int
		wantBodyStatus string
	}{
		{
			name:           "GET returns 200 with ok status",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantBodyStatus: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			rec := httptest.NewRecorder()

			handler := api.HealthHandler()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			var resp api.HealthResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}

			if resp.Status != tt.wantBodyStatus {
				t.Errorf("response status = %q, want %q", resp.Status, tt.wantBodyStatus)
			}
		})
	}
}

func TestHealthHandlerViaRouter(t *testing.T) {
	t.Parallel()

	router := api.NewRouter(nil, nil)
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("failed to GET /api/v1/health: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("body.Status = %q, want %q", body.Status, "ok")
	}
}
