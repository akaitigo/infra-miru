package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/api"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

func TestRecommendationHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		query      string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name: "returns recommendations for over-provisioned pods",
			lister: &fakePodLister{
				pods: []k8s.PodResource{
					{
						Namespace:     "default",
						PodName:       "api-pod-1",
						Deployment:    "api",
						CPURequest:    1000,
						CPUUsage:      100,
						MemoryRequest: 512 * 1024 * 1024,
						MemoryUsage:   64 * 1024 * 1024,
					},
				},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.RecommendationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Recommendations) != 1 {
					t.Fatalf("got %d recommendations, want 1", len(resp.Recommendations))
				}
				rec := resp.Recommendations[0]
				if rec.Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", rec.Deployment, "api")
				}
				if rec.MonthlySavingsJPY <= 0 {
					t.Errorf("MonthlySavingsJPY = %d, want > 0", rec.MonthlySavingsJPY)
				}
				if rec.Message == "" {
					t.Error("Message should not be empty")
				}
			},
		},
		{
			name: "no recommendations for well-provisioned pods",
			lister: &fakePodLister{
				pods: []k8s.PodResource{
					{
						Namespace:     "default",
						PodName:       "worker-pod-1",
						Deployment:    "worker",
						CPURequest:    100,
						CPUUsage:      80,
						MemoryRequest: 256 * 1024 * 1024,
						MemoryUsage:   200 * 1024 * 1024,
					},
				},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.RecommendationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Recommendations) != 0 {
					t.Errorf("got %d recommendations, want 0", len(resp.Recommendations))
				}
			},
		},
		{
			name: "empty cluster returns empty recommendations",
			lister: &fakePodLister{
				pods: []k8s.PodResource{},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.RecommendationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Recommendations) != 0 {
					t.Errorf("got %d recommendations, want 0", len(resp.Recommendations))
				}
			},
		},
		{
			name: "lister error returns 500",
			lister: &fakePodLister{
				err: fmt.Errorf("cluster unreachable"),
			},
			wantStatus: http.StatusInternalServerError,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}
				if resp.Code != "LIST_PODS_ERROR" {
					t.Errorf("error code = %q, want %q", resp.Code, "LIST_PODS_ERROR")
				}
			},
		},
		{
			name: "query parameters are accepted",
			lister: &fakePodLister{
				pods: []k8s.PodResource{},
			},
			query:      "?namespace=staging&deployment=web",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.RecommendationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := analyzer.NewAnalyzer()
			c := cost.NewCalculator()
			handler := api.RecommendationHandler(tt.lister, a, c)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			if tt.check != nil {
				tt.check(t, rec.Body.Bytes())
			}
		})
	}
}

func TestRecommendationHandlerViaRouter(t *testing.T) {
	t.Parallel()

	lister := &fakePodLister{
		pods: []k8s.PodResource{
			{
				Namespace:     "prod",
				PodName:       "api-pod-1",
				Deployment:    "api",
				CPURequest:    1000,
				CPUUsage:      100,
				MemoryRequest: 1024 * 1024 * 1024,
				MemoryUsage:   64 * 1024 * 1024,
			},
		},
	}

	router := api.NewRouter(&api.RouterDeps{
		PodLister:  lister,
		Analyzer:   analyzer.NewAnalyzer(),
		Calculator: cost.NewCalculator(),
	}, nil)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/recommendations")
	if err != nil {
		t.Fatalf("failed to GET /api/v1/recommendations: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.RecommendationResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(body.Recommendations) != 1 {
		t.Errorf("got %d recommendations, want 1", len(body.Recommendations))
	}
}
