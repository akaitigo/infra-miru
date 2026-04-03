package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/api"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// fakePodLister implements api.PodLister for testing.
type fakePodLister struct {
	pods []k8s.PodResource
	err  error
}

func (f *fakePodLister) ListPods(_ context.Context, _ k8s.ListOptions) ([]k8s.PodResource, error) {
	return f.pods, f.err
}

func TestResourceHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		query      string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name: "returns pod analyses and deployment summaries",
			lister: &fakePodLister{
				pods: []k8s.PodResource{
					{
						Namespace:     "default",
						PodName:       "api-pod-1",
						Deployment:    "api",
						CPURequest:    100,
						CPUUsage:      15,
						MemoryRequest: 256 * 1024 * 1024,
						MemoryUsage:   64 * 1024 * 1024,
					},
				},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ResourceResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Pods) != 1 {
					t.Errorf("got %d pods, want 1", len(resp.Pods))
				}
				if len(resp.Deployments) != 1 {
					t.Errorf("got %d deployments, want 1", len(resp.Deployments))
				}
				if resp.Pods[0].PodName != "api-pod-1" {
					t.Errorf("PodName = %q, want %q", resp.Pods[0].PodName, "api-pod-1")
				}
				if !resp.Pods[0].IsOverProvisioned {
					t.Error("expected IsOverProvisioned = true for 85% CPU divergence")
				}
			},
		},
		{
			name: "empty cluster returns empty arrays",
			lister: &fakePodLister{
				pods: []k8s.PodResource{},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ResourceResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Pods) != 0 {
					t.Errorf("got %d pods, want 0", len(resp.Pods))
				}
				if len(resp.Deployments) != 0 {
					t.Errorf("got %d deployments, want 0", len(resp.Deployments))
				}
			},
		},
		{
			name: "lister error returns 500",
			lister: &fakePodLister{
				err: fmt.Errorf("connection refused"),
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
			query:      "?namespace=prod&deployment=api",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ResourceResponse
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
			handler := api.ResourceHandler(tt.lister, a)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/resources"+tt.query, nil)
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

func TestResourceHandlerViaRouter(t *testing.T) {
	t.Parallel()

	lister := &fakePodLister{
		pods: []k8s.PodResource{
			{
				Namespace:     "default",
				PodName:       "web-pod-1",
				Deployment:    "web",
				CPURequest:    200,
				CPUUsage:      50,
				MemoryRequest: 512 * 1024 * 1024,
				MemoryUsage:   128 * 1024 * 1024,
			},
		},
	}

	router := api.NewRouter(&api.RouterDeps{
		PodLister:  lister,
		Analyzer:   analyzer.NewAnalyzer(),
		Calculator: nil, // not needed for resource endpoint
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/resources")
	if err != nil {
		t.Fatalf("failed to GET /api/v1/resources: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.ResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(body.Pods) != 1 {
		t.Errorf("got %d pods, want 1", len(body.Pods))
	}
}
