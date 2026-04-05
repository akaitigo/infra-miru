package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/api"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// jst is Asia/Tokyo timezone for test data construction.
var jst = time.FixedZone("JST", 9*60*60)

func TestScheduleHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		query      string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name: "returns schedule analyses for pods",
			lister: &fakePodLister{
				pods: []k8s.PodResource{
					{
						Namespace:   "default",
						PodName:     "api-pod-1",
						Deployment:  "api",
						CPUUsage:    500,
						MemoryUsage: 256 * 1024 * 1024,
						CollectedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, jst), // 10:00 JST
					},
				},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ScheduleResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Schedules) != 1 {
					t.Fatalf("got %d schedules, want 1", len(resp.Schedules))
				}
				if resp.Schedules[0].Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", resp.Schedules[0].Deployment, "api")
				}
			},
		},
		{
			name: "empty pods returns empty schedules",
			lister: &fakePodLister{
				pods: []k8s.PodResource{},
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ScheduleResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Schedules) != 0 {
					t.Errorf("got %d schedules, want 0", len(resp.Schedules))
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
				var resp api.ScheduleResponse
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
			handler := api.ScheduleHandler(tt.lister, a)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules"+tt.query, nil)
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

func TestCronHPAHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lister     *fakePodLister
		path       string
		wantStatus int
		check      func(t *testing.T, body []byte)
	}{
		{
			name: "generates CronHPA for deployment with data",
			lister: &fakePodLister{
				pods: []k8s.PodResource{
					{
						Namespace:   "prod",
						PodName:     "api-pod-1",
						Deployment:  "api",
						CPUUsage:    500,
						MemoryUsage: 256 * 1024 * 1024,
						CollectedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, jst),
					},
				},
			},
			path:       "/api/v1/cronhpa/api?namespace=prod",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.CronHPAResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.YAML == "" {
					t.Error("expected non-empty YAML")
				}
				if resp.Config.Namespace != "prod" {
					t.Errorf("Config.Namespace = %q, want %q", resp.Config.Namespace, "prod")
				}
				if resp.Config.Deployment != "api" {
					t.Errorf("Config.Deployment = %q, want %q", resp.Config.Deployment, "api")
				}
			},
		},
		{
			name: "generates default CronHPA when no matching pods",
			lister: &fakePodLister{
				pods: []k8s.PodResource{},
			},
			path:       "/api/v1/cronhpa/web?namespace=staging",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.CronHPAResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Config.ScaleDownTime != "22:00" {
					t.Errorf("ScaleDownTime = %q, want default %q", resp.Config.ScaleDownTime, "22:00")
				}
				if resp.Config.ScaleUpTime != "06:00" {
					t.Errorf("ScaleUpTime = %q, want default %q", resp.Config.ScaleUpTime, "06:00")
				}
			},
		},
		{
			name: "missing namespace returns 400",
			lister: &fakePodLister{
				pods: []k8s.PodResource{},
			},
			path:       "/api/v1/cronhpa/api",
			wantStatus: http.StatusBadRequest,
			check: func(t *testing.T, body []byte) {
				t.Helper()
				var resp api.ErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}
				if resp.Code != "MISSING_NAMESPACE" {
					t.Errorf("error code = %q, want %q", resp.Code, "MISSING_NAMESPACE")
				}
			},
		},
		{
			name: "lister error returns 500",
			lister: &fakePodLister{
				err: fmt.Errorf("cluster unreachable"),
			},
			path:       "/api/v1/cronhpa/api?namespace=prod",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := api.NewRouter(&api.RouterDeps{
				PodLister:  tt.lister,
				Analyzer:   analyzer.NewAnalyzer(),
				Calculator: cost.NewCalculator(),
			}, nil)

			srv := httptest.NewServer(router)
			defer srv.Close()

			resp, err := http.Get(srv.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to GET %s: %v", tt.path, err)
			}
			defer func() {
				if closeErr := resp.Body.Close(); closeErr != nil {
					t.Logf("failed to close response body: %v", closeErr)
				}
			}()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			var body []byte
			body, err = readBody(resp)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			if tt.check != nil {
				tt.check(t, body)
			}
		})
	}
}

func TestScheduleHandlerViaRouter(t *testing.T) {
	t.Parallel()

	lister := &fakePodLister{
		pods: []k8s.PodResource{
			{
				Namespace:   "default",
				PodName:     "api-pod-1",
				Deployment:  "api",
				CPUUsage:    500,
				MemoryUsage: 256 * 1024 * 1024,
				CollectedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, jst),
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

	resp, err := http.Get(srv.URL + "/api/v1/schedules")
	if err != nil {
		t.Fatalf("failed to GET /api/v1/schedules: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(body.Schedules) != 1 {
		t.Errorf("got %d schedules, want 1", len(body.Schedules))
	}
}

// readBody reads all bytes from an HTTP response body.
func readBody(resp *http.Response) ([]byte, error) {
	var buf []byte
	buf, err := readAllFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// readAllFrom is a small helper to read all bytes from an io.Reader.
func readAllFrom(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}
	return result, nil
}
