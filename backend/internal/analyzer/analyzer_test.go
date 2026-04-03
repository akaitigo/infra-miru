package analyzer_test

import (
	"math"
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

const floatEpsilon = 0.01

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < floatEpsilon
}

func TestAnalyzeResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		pods  []k8s.PodResource
		check func(t *testing.T, results []analyzer.ResourceAnalysis)
	}{
		{
			name: "high divergence marks over-provisioned",
			pods: []k8s.PodResource{
				{
					Namespace:     "default",
					PodName:       "api-abc-123",
					Deployment:    "api",
					CPURequest:    100, // 100m
					CPUUsage:      15,  // 15m → 85% divergence
					MemoryRequest: 256 * 1024 * 1024,
					MemoryUsage:   64 * 1024 * 1024, // 75% divergence
				},
			},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 1 {
					t.Fatalf("got %d results, want 1", len(results))
				}
				r := results[0]

				if !almostEqual(r.CPUDivergence, 85.0) {
					t.Errorf("CPUDivergence = %.2f, want 85.00", r.CPUDivergence)
				}
				if !almostEqual(r.MemoryDivergence, 75.0) {
					t.Errorf("MemoryDivergence = %.2f, want 75.00", r.MemoryDivergence)
				}
				if !r.IsOverProvisioned {
					t.Error("expected IsOverProvisioned = true")
				}
			},
		},
		{
			name: "low divergence is not over-provisioned",
			pods: []k8s.PodResource{
				{
					Namespace:     "default",
					PodName:       "worker-def-456",
					Deployment:    "worker",
					CPURequest:    100,
					CPUUsage:      80, // 20% divergence
					MemoryRequest: 256 * 1024 * 1024,
					MemoryUsage:   200 * 1024 * 1024, // ~21.9% divergence
				},
			},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 1 {
					t.Fatalf("got %d results, want 1", len(results))
				}
				r := results[0]

				if !almostEqual(r.CPUDivergence, 20.0) {
					t.Errorf("CPUDivergence = %.2f, want 20.00", r.CPUDivergence)
				}
				if r.IsOverProvisioned {
					t.Error("expected IsOverProvisioned = false")
				}
			},
		},
		{
			name: "exact boundary at 50 percent is over-provisioned",
			pods: []k8s.PodResource{
				{
					Namespace:     "default",
					PodName:       "edge-case-pod",
					Deployment:    "edge",
					CPURequest:    100,
					CPUUsage:      50, // exactly 50% divergence
					MemoryRequest: 100,
					MemoryUsage:   100, // 0% divergence
				},
			},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 1 {
					t.Fatalf("got %d results, want 1", len(results))
				}
				r := results[0]

				if !almostEqual(r.CPUDivergence, 50.0) {
					t.Errorf("CPUDivergence = %.2f, want 50.00", r.CPUDivergence)
				}
				if !r.IsOverProvisioned {
					t.Error("expected IsOverProvisioned = true at 50% boundary")
				}
			},
		},
		{
			name: "zero request results in zero divergence",
			pods: []k8s.PodResource{
				{
					Namespace:     "default",
					PodName:       "no-request-pod",
					Deployment:    "no-req",
					CPURequest:    0,
					CPUUsage:      50,
					MemoryRequest: 0,
					MemoryUsage:   1024,
				},
			},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 1 {
					t.Fatalf("got %d results, want 1", len(results))
				}
				r := results[0]

				if r.CPUDivergence != 0 {
					t.Errorf("CPUDivergence = %.2f, want 0 for zero request", r.CPUDivergence)
				}
				if r.MemoryDivergence != 0 {
					t.Errorf("MemoryDivergence = %.2f, want 0 for zero request", r.MemoryDivergence)
				}
				if r.IsOverProvisioned {
					t.Error("expected IsOverProvisioned = false for zero request")
				}
			},
		},
		{
			name: "usage exceeds request gives negative divergence",
			pods: []k8s.PodResource{
				{
					Namespace:     "default",
					PodName:       "over-usage-pod",
					Deployment:    "heavy",
					CPURequest:    100,
					CPUUsage:      150, // -50% divergence (under-provisioned)
					MemoryRequest: 100,
					MemoryUsage:   200,
				},
			},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 1 {
					t.Fatalf("got %d results, want 1", len(results))
				}
				r := results[0]

				if !almostEqual(r.CPUDivergence, -50.0) {
					t.Errorf("CPUDivergence = %.2f, want -50.00", r.CPUDivergence)
				}
				if r.IsOverProvisioned {
					t.Error("expected IsOverProvisioned = false for under-provisioned pods")
				}
			},
		},
		{
			name: "multiple pods are all analyzed",
			pods: []k8s.PodResource{
				{
					Namespace:     "ns-a",
					PodName:       "pod-1",
					Deployment:    "app",
					CPURequest:    200,
					CPUUsage:      40, // 80% divergence
					MemoryRequest: 512 * 1024 * 1024,
					MemoryUsage:   128 * 1024 * 1024,
				},
				{
					Namespace:     "ns-b",
					PodName:       "pod-2",
					Deployment:    "web",
					CPURequest:    100,
					CPUUsage:      90, // 10% divergence
					MemoryRequest: 256 * 1024 * 1024,
					MemoryUsage:   240 * 1024 * 1024,
				},
			},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 2 {
					t.Fatalf("got %d results, want 2", len(results))
				}
				if !results[0].IsOverProvisioned {
					t.Error("pod-1 should be over-provisioned")
				}
				if results[1].IsOverProvisioned {
					t.Error("pod-2 should not be over-provisioned")
				}
			},
		},
		{
			name: "empty input returns empty result",
			pods: []k8s.PodResource{},
			check: func(t *testing.T, results []analyzer.ResourceAnalysis) {
				t.Helper()
				if len(results) != 0 {
					t.Fatalf("got %d results, want 0", len(results))
				}
			},
		},
	}

	a := analyzer.NewAnalyzer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := a.AnalyzeResources(tt.pods)
			tt.check(t, results)
		})
	}
}

func TestAggregateByDeployment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		analyses []analyzer.ResourceAnalysis
		check    func(t *testing.T, summaries []analyzer.DeploymentSummary)
	}{
		{
			name: "aggregate multiple pods in one deployment",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "default",
					PodName:           "api-pod-1",
					Deployment:        "api",
					CPURequest:        100,
					CPUUsage:          20,
					CPUDivergence:     80.0,
					MemoryRequest:     256 * 1024 * 1024,
					MemoryUsage:       64 * 1024 * 1024,
					MemoryDivergence:  75.0,
					IsOverProvisioned: true,
				},
				{
					Namespace:         "default",
					PodName:           "api-pod-2",
					Deployment:        "api",
					CPURequest:        100,
					CPUUsage:          30,
					CPUDivergence:     70.0,
					MemoryRequest:     256 * 1024 * 1024,
					MemoryUsage:       128 * 1024 * 1024,
					MemoryDivergence:  50.0,
					IsOverProvisioned: true,
				},
			},
			check: func(t *testing.T, summaries []analyzer.DeploymentSummary) {
				t.Helper()
				if len(summaries) != 1 {
					t.Fatalf("got %d summaries, want 1", len(summaries))
				}
				s := summaries[0]
				if s.Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", s.Deployment, "api")
				}
				if s.PodCount != 2 {
					t.Errorf("PodCount = %d, want 2", s.PodCount)
				}
				if s.TotalCPURequest != 200 {
					t.Errorf("TotalCPURequest = %d, want 200", s.TotalCPURequest)
				}
				if s.TotalCPUUsage != 50 {
					t.Errorf("TotalCPUUsage = %d, want 50", s.TotalCPUUsage)
				}
				if !almostEqual(s.AvgCPUDivergence, 75.0) {
					t.Errorf("AvgCPUDivergence = %.2f, want 75.00", s.AvgCPUDivergence)
				}
				if !almostEqual(s.AvgMemoryDivergence, 62.5) {
					t.Errorf("AvgMemoryDivergence = %.2f, want 62.50", s.AvgMemoryDivergence)
				}
				if !s.IsOverProvisioned {
					t.Error("expected IsOverProvisioned = true")
				}
			},
		},
		{
			name: "separate deployments produce separate summaries",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "default",
					PodName:           "api-pod-1",
					Deployment:        "api",
					CPURequest:        100,
					CPUUsage:          10,
					CPUDivergence:     90.0,
					MemoryRequest:     100,
					MemoryUsage:       10,
					MemoryDivergence:  90.0,
					IsOverProvisioned: true,
				},
				{
					Namespace:         "default",
					PodName:           "web-pod-1",
					Deployment:        "web",
					CPURequest:        100,
					CPUUsage:          90,
					CPUDivergence:     10.0,
					MemoryRequest:     100,
					MemoryUsage:       95,
					MemoryDivergence:  5.0,
					IsOverProvisioned: false,
				},
			},
			check: func(t *testing.T, summaries []analyzer.DeploymentSummary) {
				t.Helper()
				if len(summaries) != 2 {
					t.Fatalf("got %d summaries, want 2", len(summaries))
				}
				// Order should be preserved (api first, web second).
				if summaries[0].Deployment != "api" {
					t.Errorf("first summary Deployment = %q, want %q", summaries[0].Deployment, "api")
				}
				if summaries[1].Deployment != "web" {
					t.Errorf("second summary Deployment = %q, want %q", summaries[1].Deployment, "web")
				}
				if !summaries[0].IsOverProvisioned {
					t.Error("api should be over-provisioned")
				}
				if summaries[1].IsOverProvisioned {
					t.Error("web should not be over-provisioned")
				}
			},
		},
		{
			name:     "empty input returns empty result",
			analyses: []analyzer.ResourceAnalysis{},
			check: func(t *testing.T, summaries []analyzer.DeploymentSummary) {
				t.Helper()
				if len(summaries) != 0 {
					t.Fatalf("got %d summaries, want 0", len(summaries))
				}
			},
		},
		{
			name: "same deployment in different namespaces are separate",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "prod",
					PodName:           "api-pod-1",
					Deployment:        "api",
					CPURequest:        100,
					CPUUsage:          10,
					CPUDivergence:     90.0,
					MemoryRequest:     100,
					MemoryUsage:       10,
					MemoryDivergence:  90.0,
					IsOverProvisioned: true,
				},
				{
					Namespace:         "staging",
					PodName:           "api-pod-1",
					Deployment:        "api",
					CPURequest:        100,
					CPUUsage:          80,
					CPUDivergence:     20.0,
					MemoryRequest:     100,
					MemoryUsage:       80,
					MemoryDivergence:  20.0,
					IsOverProvisioned: false,
				},
			},
			check: func(t *testing.T, summaries []analyzer.DeploymentSummary) {
				t.Helper()
				if len(summaries) != 2 {
					t.Fatalf("got %d summaries, want 2", len(summaries))
				}
				if summaries[0].Namespace != "prod" {
					t.Errorf("first summary Namespace = %q, want %q", summaries[0].Namespace, "prod")
				}
				if summaries[1].Namespace != "staging" {
					t.Errorf("second summary Namespace = %q, want %q", summaries[1].Namespace, "staging")
				}
			},
		},
	}

	a := analyzer.NewAnalyzer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			summaries := a.AggregateByDeployment(tt.analyses)
			tt.check(t, summaries)
		})
	}
}
