package cost_test

import (
	"strings"
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
)

func TestCalculateRecommendations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		analyses []analyzer.ResourceAnalysis
		check    func(t *testing.T, recs []cost.Recommendation)
	}{
		{
			name: "over-provisioned pod generates recommendation with correct savings",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "default",
					PodName:           "api-pod-1",
					Deployment:        "api",
					CPURequest:        1000, // 1000m
					CPUUsage:          100,  // 100m → 90% divergence
					CPUDivergence:     90.0,
					MemoryRequest:     512 * 1024 * 1024, // 512MB
					MemoryUsage:       64 * 1024 * 1024,  // 64MB → 87.5% divergence
					MemoryDivergence:  87.5,
					IsOverProvisioned: true,
				},
			},
			check: func(t *testing.T, recs []cost.Recommendation) {
				t.Helper()
				if len(recs) != 1 {
					t.Fatalf("got %d recommendations, want 1", len(recs))
				}
				r := recs[0]

				if r.Namespace != "default" {
					t.Errorf("Namespace = %q, want %q", r.Namespace, "default")
				}
				if r.Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", r.Deployment, "api")
				}

				// Recommended CPU = 100 * 1.2 = 120
				if r.RecommendedCPU != 120 {
					t.Errorf("RecommendedCPU = %d, want 120", r.RecommendedCPU)
				}

				// Recommended Memory = 64MB * 1.2 = 76.8MB ≈ 80530636 bytes
				usageMem := int64(64 * 1024 * 1024)
				expectedMemBytes := int64(float64(usageMem) * 1.2)
				if r.RecommendedMemory != expectedMemBytes {
					t.Errorf("RecommendedMemory = %d, want %d", r.RecommendedMemory, expectedMemBytes)
				}

				// CPU savings: (1000 - 120) * 3.5 = 3080
				// Memory savings: ((512MB - 76.8MB) / 1MB) * 0.5
				//   = (512*1024*1024 - 80530636) / (1024*1024) * 0.5
				//   = (536870912 - 80530636) / 1048576 * 0.5
				//   = 435.2 * 0.5 = 217.6
				// Total ≈ 3080 + 217 = 3297
				if r.MonthlySavingsJPY <= 0 {
					t.Errorf("MonthlySavingsJPY = %d, want > 0", r.MonthlySavingsJPY)
				}

				if r.Message == "" {
					t.Error("Message should not be empty")
				}
				if !strings.Contains(r.Message, "節約") {
					t.Errorf("Message should contain '節約', got: %s", r.Message)
				}
			},
		},
		{
			name: "under 50 percent divergence is skipped",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "default",
					PodName:           "worker-pod-1",
					Deployment:        "worker",
					CPURequest:        100,
					CPUUsage:          80, // 20% divergence
					CPUDivergence:     20.0,
					MemoryRequest:     256 * 1024 * 1024,
					MemoryUsage:       200 * 1024 * 1024, // ~21.9% divergence
					MemoryDivergence:  21.875,
					IsOverProvisioned: false,
				},
			},
			check: func(t *testing.T, recs []cost.Recommendation) {
				t.Helper()
				if len(recs) != 0 {
					t.Fatalf("got %d recommendations, want 0 for low-divergence pod", len(recs))
				}
			},
		},
		{
			name: "multiple pods in same deployment are aggregated",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "prod",
					PodName:           "api-pod-1",
					Deployment:        "api",
					CPURequest:        500,
					CPUUsage:          50,
					CPUDivergence:     90.0,
					MemoryRequest:     256 * 1024 * 1024,
					MemoryUsage:       25 * 1024 * 1024,
					MemoryDivergence:  90.234375,
					IsOverProvisioned: true,
				},
				{
					Namespace:         "prod",
					PodName:           "api-pod-2",
					Deployment:        "api",
					CPURequest:        500,
					CPUUsage:          100,
					CPUDivergence:     80.0,
					MemoryRequest:     256 * 1024 * 1024,
					MemoryUsage:       50 * 1024 * 1024,
					MemoryDivergence:  80.46875,
					IsOverProvisioned: true,
				},
			},
			check: func(t *testing.T, recs []cost.Recommendation) {
				t.Helper()
				if len(recs) != 1 {
					t.Fatalf("got %d recommendations, want 1 (aggregated)", len(recs))
				}
				r := recs[0]
				if r.Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", r.Deployment, "api")
				}
				// Total CPU request = 1000, total CPU usage = 150
				// Recommended CPU = 150 * 1.2 = 180
				if r.RecommendedCPU != 180 {
					t.Errorf("RecommendedCPU = %d, want 180", r.RecommendedCPU)
				}
				if r.CurrentRequestCPU != 1000 {
					t.Errorf("CurrentRequestCPU = %d, want 1000", r.CurrentRequestCPU)
				}
				if r.MonthlySavingsJPY <= 0 {
					t.Errorf("MonthlySavingsJPY = %d, want > 0", r.MonthlySavingsJPY)
				}
			},
		},
		{
			name:     "empty input returns empty result",
			analyses: []analyzer.ResourceAnalysis{},
			check: func(t *testing.T, recs []cost.Recommendation) {
				t.Helper()
				if len(recs) != 0 {
					t.Fatalf("got %d recommendations, want 0", len(recs))
				}
			},
		},
		{
			name: "mixed over and under provisioned pods",
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
					MemoryUsage:       90,
					MemoryDivergence:  10.0,
					IsOverProvisioned: false,
				},
			},
			check: func(t *testing.T, recs []cost.Recommendation) {
				t.Helper()
				if len(recs) != 1 {
					t.Fatalf("got %d recommendations, want 1 (only over-provisioned)", len(recs))
				}
				if recs[0].Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", recs[0].Deployment, "api")
				}
			},
		},
		{
			name: "zero usage pod gets zero recommended resources",
			analyses: []analyzer.ResourceAnalysis{
				{
					Namespace:         "default",
					PodName:           "idle-pod-1",
					Deployment:        "idle",
					CPURequest:        1000,
					CPUUsage:          0,
					CPUDivergence:     100.0,
					MemoryRequest:     1024 * 1024 * 1024,
					MemoryUsage:       0,
					MemoryDivergence:  100.0,
					IsOverProvisioned: true,
				},
			},
			check: func(t *testing.T, recs []cost.Recommendation) {
				t.Helper()
				if len(recs) != 1 {
					t.Fatalf("got %d recommendations, want 1", len(recs))
				}
				r := recs[0]
				if r.RecommendedCPU != 0 {
					t.Errorf("RecommendedCPU = %d, want 0 for idle pod", r.RecommendedCPU)
				}
				if r.RecommendedMemory != 0 {
					t.Errorf("RecommendedMemory = %d, want 0 for idle pod", r.RecommendedMemory)
				}
				if r.MonthlySavingsJPY <= 0 {
					t.Errorf("MonthlySavingsJPY = %d, want > 0", r.MonthlySavingsJPY)
				}
			},
		},
	}

	calc := cost.NewCalculator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recs := calc.CalculateRecommendations(tt.analyses)
			tt.check(t, recs)
		})
	}
}

func TestNewCalculatorWithPricing(t *testing.T) {
	t.Parallel()

	// Custom pricing: CPU ¥10/millicore/month, Memory ¥1/MB/month
	calc := cost.NewCalculatorWithPricing(10.0, 1.0)

	analyses := []analyzer.ResourceAnalysis{
		{
			Namespace:         "default",
			PodName:           "api-pod-1",
			Deployment:        "api",
			CPURequest:        1000,
			CPUUsage:          100,
			CPUDivergence:     90.0,
			MemoryRequest:     512 * 1024 * 1024,
			MemoryUsage:       64 * 1024 * 1024,
			MemoryDivergence:  87.5,
			IsOverProvisioned: true,
		},
	}

	recs := calc.CalculateRecommendations(analyses)
	if len(recs) != 1 {
		t.Fatalf("got %d recommendations, want 1", len(recs))
	}

	// CPU savings: (1000 - 120) * 10 = 8800
	// Memory savings much higher with custom pricing.
	defaultCalc := cost.NewCalculator()
	defaultRecs := defaultCalc.CalculateRecommendations(analyses)

	if recs[0].MonthlySavingsJPY <= defaultRecs[0].MonthlySavingsJPY {
		t.Errorf(
			"custom pricing savings (%d) should be > default pricing savings (%d)",
			recs[0].MonthlySavingsJPY, defaultRecs[0].MonthlySavingsJPY,
		)
	}
}
