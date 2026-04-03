package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/api"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// httpResult holds the result of an HTTP request with the body already read and
// closed. This avoids bodyclose linter warnings by ensuring the response body
// is always consumed and closed inside doGet.
type httpResult struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

// integrationServer creates an httptest.Server with all routes registered using
// the given fakePodLister. The caller must call srv.Close() when done.
func integrationServer(lister *fakePodLister) *httptest.Server {
	router := api.NewRouter(&api.RouterDeps{
		PodLister:  lister,
		Analyzer:   analyzer.NewAnalyzer(),
		Calculator: cost.NewCalculator(),
	})
	return httptest.NewServer(router)
}

// overProvisionedPods returns a slice of pods where CPU/memory usage is much
// lower than requests, causing them to be flagged as over-provisioned.
func overProvisionedPods() []k8s.PodResource {
	jst := time.FixedZone("JST", 9*60*60)
	return []k8s.PodResource{
		{
			Namespace:     "default",
			PodName:       "web-pod-1",
			Deployment:    "web",
			CPURequest:    1000,
			CPUUsage:      100,
			MemoryRequest: 1024 * 1024 * 1024,
			MemoryUsage:   64 * 1024 * 1024,
			CollectedAt:   time.Date(2026, 4, 1, 1, 0, 0, 0, jst),
		},
		{
			Namespace:     "default",
			PodName:       "web-pod-2",
			Deployment:    "web",
			CPURequest:    1000,
			CPUUsage:      120,
			MemoryRequest: 1024 * 1024 * 1024,
			MemoryUsage:   80 * 1024 * 1024,
			CollectedAt:   time.Date(2026, 4, 1, 2, 0, 0, 0, jst),
		},
	}
}

// wellProvisionedPods returns a slice of pods where usage is close to requests.
func wellProvisionedPods() []k8s.PodResource {
	jst := time.FixedZone("JST", 9*60*60)
	return []k8s.PodResource{
		{
			Namespace:     "production",
			PodName:       "api-pod-1",
			Deployment:    "api",
			CPURequest:    100,
			CPUUsage:      80,
			MemoryRequest: 256 * 1024 * 1024,
			MemoryUsage:   200 * 1024 * 1024,
			CollectedAt:   time.Date(2026, 4, 1, 10, 0, 0, 0, jst),
		},
	}
}

// doGet performs a GET request against the given URL with a background context.
// It reads and closes the response body, returning the result in an httpResult.
func doGet(t *testing.T, url string) httpResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("failed to close response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return httpResult{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
	}
}

func TestIntegration_Health(t *testing.T) {
	t.Parallel()

	srv := integrationServer(&fakePodLister{pods: []k8s.PodResource{}})
	defer srv.Close()

	result := doGet(t, srv.URL+"/api/v1/health")

	if result.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}

	if result.ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", result.ContentType, "application/json")
	}

	var health api.HealthResponse
	if err := json.Unmarshal(result.Body, &health); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}
	if health.Status != "ok" {
		t.Errorf("health.Status = %q, want %q", health.Status, "ok")
	}
}

func TestIntegration_Resources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		path       string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:       "returns pods and deployments",
			lister:     &fakePodLister{pods: overProvisionedPods()},
			path:       "/api/v1/resources",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ResourceResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(resp.Pods) != 2 {
					t.Errorf("got %d pods, want 2", len(resp.Pods))
				}
				if len(resp.Deployments) != 1 {
					t.Errorf("got %d deployments, want 1", len(resp.Deployments))
				}
				if resp.Deployments[0].Deployment != "web" {
					t.Errorf("Deployment = %q, want %q", resp.Deployments[0].Deployment, "web")
				}
			},
		},
		{
			name:       "namespace filter is accepted",
			lister:     &fakePodLister{pods: overProvisionedPods()},
			path:       "/api/v1/resources?namespace=default",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ResourceResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				// fakePodLister ignores filters, so we just verify the endpoint
				// processes the parameter without error.
				if len(resp.Pods) == 0 {
					t.Error("expected pods in response")
				}
			},
		},
		{
			name:       "empty cluster returns empty arrays",
			lister:     &fakePodLister{pods: []k8s.PodResource{}},
			path:       "/api/v1/resources",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ResourceResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(resp.Pods) != 0 {
					t.Errorf("got %d pods, want 0", len(resp.Pods))
				}
				if len(resp.Deployments) != 0 {
					t.Errorf("got %d deployments, want 0", len(resp.Deployments))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := integrationServer(tt.lister)
			defer srv.Close()

			result := doGet(t, srv.URL+tt.path)

			if result.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", result.StatusCode, tt.wantStatus)
			}

			if result.ContentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", result.ContentType, "application/json")
			}

			if tt.check != nil {
				tt.check(t, result.Body)
			}
		})
	}
}

func TestIntegration_Recommendations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:       "over-provisioned pods generate recommendations",
			lister:     &fakePodLister{pods: overProvisionedPods()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.RecommendationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(resp.Recommendations) == 0 {
					t.Error("expected at least one recommendation for over-provisioned pods")
				}
				for _, rec := range resp.Recommendations {
					if rec.Deployment == "" {
						t.Error("Deployment should not be empty")
					}
					if rec.MonthlySavingsJPY <= 0 {
						t.Errorf("MonthlySavingsJPY = %d, want > 0", rec.MonthlySavingsJPY)
					}
					if rec.Message == "" {
						t.Error("Message should not be empty")
					}
				}
			},
		},
		{
			name:       "well-provisioned pods return no recommendations",
			lister:     &fakePodLister{pods: wellProvisionedPods()},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.RecommendationResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(resp.Recommendations) != 0 {
					t.Errorf("got %d recommendations, want 0 for well-provisioned pods", len(resp.Recommendations))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := integrationServer(tt.lister)
			defer srv.Close()

			result := doGet(t, srv.URL+"/api/v1/recommendations")

			if result.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", result.StatusCode, tt.wantStatus)
			}

			if result.ContentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", result.ContentType, "application/json")
			}

			if tt.check != nil {
				tt.check(t, result.Body)
			}
		})
	}
}

func TestIntegration_Schedules(t *testing.T) {
	t.Parallel()

	srv := integrationServer(&fakePodLister{pods: overProvisionedPods()})
	defer srv.Close()

	result := doGet(t, srv.URL+"/api/v1/schedules")

	if result.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}

	if result.ContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", result.ContentType, "application/json")
	}

	var schedResp api.ScheduleResponse
	if err := json.Unmarshal(result.Body, &schedResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(schedResp.Schedules) == 0 {
		t.Error("expected at least one schedule for pods with data")
	}
}

func TestIntegration_CronHPA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		path       string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name:       "generates YAML for deployment",
			lister:     &fakePodLister{pods: overProvisionedPods()},
			path:       "/api/v1/cronhpa/web?namespace=default",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.CronHPAResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if resp.YAML == "" {
					t.Error("expected non-empty YAML")
				}
				if resp.Config.Namespace != "default" {
					t.Errorf("Config.Namespace = %q, want %q", resp.Config.Namespace, "default")
				}
				if resp.Config.Deployment != "web" {
					t.Errorf("Config.Deployment = %q, want %q", resp.Config.Deployment, "web")
				}
			},
		},
		{
			name:       "missing namespace returns 400",
			lister:     &fakePodLister{pods: []k8s.PodResource{}},
			path:       "/api/v1/cronhpa/my-app",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if resp.Code != "MISSING_NAMESPACE" {
					t.Errorf("code = %q, want %q", resp.Code, "MISSING_NAMESPACE")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := integrationServer(tt.lister)
			defer srv.Close()

			result := doGet(t, srv.URL+tt.path)

			if result.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", result.StatusCode, tt.wantStatus)
			}

			if result.ContentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", result.ContentType, "application/json")
			}

			if tt.check != nil {
				tt.check(t, result.Body)
			}
		})
	}
}

func TestIntegration_UnknownPath(t *testing.T) {
	t.Parallel()

	srv := integrationServer(&fakePodLister{pods: []k8s.PodResource{}})
	defer srv.Close()

	result := doGet(t, srv.URL+"/api/v1/unknown")

	// chi returns 405 Method Not Allowed for known route prefixes and 404 for
	// completely unknown paths. Either indicates the path is not served.
	if result.StatusCode != http.StatusNotFound && result.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 404 or 405", result.StatusCode)
	}
}

func TestIntegration_JSONContentType(t *testing.T) {
	t.Parallel()

	srv := integrationServer(&fakePodLister{pods: overProvisionedPods()})
	defer srv.Close()

	endpoints := []string{
		"/api/v1/health",
		"/api/v1/resources",
		"/api/v1/recommendations",
		"/api/v1/schedules",
		"/api/v1/cronhpa/web?namespace=default",
	}

	for _, ep := range endpoints {
		result := doGet(t, srv.URL+ep)

		if result.ContentType != "application/json" {
			t.Errorf("Content-Type for %s = %q, want %q", ep, result.ContentType, "application/json")
		}
	}
}
