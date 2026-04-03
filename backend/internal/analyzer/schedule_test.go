package analyzer_test

import (
	"testing"
	"time"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// jst is Asia/Tokyo timezone for test data construction.
var jst = time.FixedZone("JST", 9*60*60)

func TestAnalyzeSchedule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		pods              []k8s.PodResource
		wantCount         int
		checkDeployment   string
		checkLowLoadHours []int
		checkWeekendLow   bool
	}{
		{
			name:      "empty pods returns empty result",
			pods:      []k8s.PodResource{},
			wantCount: 0,
		},
		{
			name: "pods without deployment are skipped",
			pods: []k8s.PodResource{
				{
					Namespace:   "default",
					PodName:     "orphan-pod",
					Deployment:  "",
					CPUUsage:    100,
					MemoryUsage: 256 * 1024 * 1024,
					CollectedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, jst),
				},
			},
			wantCount: 0,
		},
		{
			name: "nighttime low load detection",
			pods: nighttimeLowLoadPods(),
			// Daytime pods at hour 10,14 with CPU 1000; nighttime at hour 2 with CPU 10.
			// Overall avg ~670. Threshold = 670*0.30 = ~201. Hour 2 (cpu=10) is below.
			wantCount:         1,
			checkDeployment:   "api",
			checkLowLoadHours: []int{2},
			checkWeekendLow:   false,
		},
		{
			name:              "weekend low load detection",
			pods:              weekendLowLoadPods(),
			wantCount:         1,
			checkDeployment:   "api",
			checkLowLoadHours: nil, // all hours have similar load
			checkWeekendLow:   true,
		},
		{
			name: "threshold boundary - exactly at 30% is not low load",
			pods: thresholdBoundaryPods(),
			// Two hours: hour 10 with CPU 1000, hour 2 with CPU 300.
			// Overall avg = (1000+300)/2 = 650. Threshold = 650*0.30 = 195.
			// Hour 2 (300) > 195, so not low-load.
			wantCount:         1,
			checkDeployment:   "api",
			checkLowLoadHours: nil,
			checkWeekendLow:   false,
		},
		{
			name: "multiple deployments analyzed separately",
			pods: multiDeploymentPods(),
			// Two deployments: "api" and "worker"
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := analyzer.NewAnalyzer()
			results := a.AnalyzeSchedule(tt.pods)

			if len(results) != tt.wantCount {
				t.Fatalf("got %d schedule analyses, want %d", len(results), tt.wantCount)
			}

			if tt.checkDeployment == "" {
				return
			}

			var found *analyzer.ScheduleAnalysis
			for i := range results {
				if results[i].Deployment == tt.checkDeployment {
					found = &results[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("deployment %q not found in results", tt.checkDeployment)
			}

			if tt.checkLowLoadHours != nil {
				if !intSliceEqual(found.LowLoadHours, tt.checkLowLoadHours) {
					t.Errorf("LowLoadHours = %v, want %v", found.LowLoadHours, tt.checkLowLoadHours)
				}
			} else {
				if len(found.LowLoadHours) != 0 {
					t.Errorf("LowLoadHours = %v, want empty", found.LowLoadHours)
				}
			}

			if found.IsWeekendLowLoad != tt.checkWeekendLow {
				t.Errorf("IsWeekendLowLoad = %v, want %v", found.IsWeekendLowLoad, tt.checkWeekendLow)
			}
		})
	}
}

func TestAnalyzeScheduleHourlyLoads(t *testing.T) {
	t.Parallel()

	// Create pods at two distinct hours.
	pods := []k8s.PodResource{
		{
			Namespace:   "default",
			PodName:     "api-pod-1",
			Deployment:  "api",
			CPUUsage:    100,
			MemoryUsage: 256 * 1024 * 1024,
			CollectedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, jst), // hour 10 JST
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-2",
			Deployment:  "api",
			CPUUsage:    200,
			MemoryUsage: 512 * 1024 * 1024,
			CollectedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, jst), // same hour
		},
	}

	a := analyzer.NewAnalyzer()
	results := a.AnalyzeSchedule(pods)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	if len(results[0].HourlyLoads) != 1 {
		t.Fatalf("got %d hourly loads, want 1", len(results[0].HourlyLoads))
	}

	hl := results[0].HourlyLoads[0]
	if hl.SampleCount != 2 {
		t.Errorf("SampleCount = %d, want 2", hl.SampleCount)
	}

	// Average CPU: (100+200)/2 = 150
	wantAvgCPU := 150.0
	if hl.AvgCPUUsage != wantAvgCPU {
		t.Errorf("AvgCPUUsage = %f, want %f", hl.AvgCPUUsage, wantAvgCPU)
	}
}

// nighttimeLowLoadPods creates pods with high daytime and low nighttime usage.
func nighttimeLowLoadPods() []k8s.PodResource {
	// Tuesday 2026-03-31
	day := time.Date(2026, 3, 31, 0, 0, 0, 0, jst)

	return []k8s.PodResource{
		{
			Namespace:   "default",
			PodName:     "api-pod-1",
			Deployment:  "api",
			CPUUsage:    1000,
			MemoryUsage: 512 * 1024 * 1024,
			CollectedAt: day.Add(10 * time.Hour), // 10:00 JST
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-2",
			Deployment:  "api",
			CPUUsage:    1000,
			MemoryUsage: 512 * 1024 * 1024,
			CollectedAt: day.Add(14 * time.Hour), // 14:00 JST
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-3",
			Deployment:  "api",
			CPUUsage:    10, // very low
			MemoryUsage: 32 * 1024 * 1024,
			CollectedAt: day.Add(2 * time.Hour), // 02:00 JST (nighttime)
		},
	}
}

// weekendLowLoadPods creates pods with high weekday and very low weekend usage.
func weekendLowLoadPods() []k8s.PodResource {
	// Monday 2026-03-30
	weekday := time.Date(2026, 3, 30, 10, 0, 0, 0, jst)
	// Saturday 2026-04-04
	weekend := time.Date(2026, 4, 4, 10, 0, 0, 0, jst)

	return []k8s.PodResource{
		{
			Namespace:   "default",
			PodName:     "api-pod-1",
			Deployment:  "api",
			CPUUsage:    1000,
			MemoryUsage: 512 * 1024 * 1024,
			CollectedAt: weekday,
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-2",
			Deployment:  "api",
			CPUUsage:    900,
			MemoryUsage: 480 * 1024 * 1024,
			CollectedAt: weekday.Add(time.Hour),
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-3",
			Deployment:  "api",
			CPUUsage:    50, // much lower than weekday
			MemoryUsage: 32 * 1024 * 1024,
			CollectedAt: weekend,
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-4",
			Deployment:  "api",
			CPUUsage:    40,
			MemoryUsage: 28 * 1024 * 1024,
			CollectedAt: weekend.Add(time.Hour),
		},
	}
}

// thresholdBoundaryPods creates pods at exactly the boundary of the low load threshold.
func thresholdBoundaryPods() []k8s.PodResource {
	day := time.Date(2026, 3, 31, 0, 0, 0, 0, jst) // Tuesday

	return []k8s.PodResource{
		{
			Namespace:   "default",
			PodName:     "api-pod-1",
			Deployment:  "api",
			CPUUsage:    1000,
			MemoryUsage: 512 * 1024 * 1024,
			CollectedAt: day.Add(10 * time.Hour), // 10:00 JST
		},
		{
			Namespace:   "default",
			PodName:     "api-pod-2",
			Deployment:  "api",
			CPUUsage:    300, // 300 > 650*0.30=195, so not low-load
			MemoryUsage: 128 * 1024 * 1024,
			CollectedAt: day.Add(2 * time.Hour), // 02:00 JST
		},
	}
}

// multiDeploymentPods creates pods from two different deployments.
func multiDeploymentPods() []k8s.PodResource {
	day := time.Date(2026, 3, 31, 10, 0, 0, 0, jst)

	return []k8s.PodResource{
		{
			Namespace:   "default",
			PodName:     "api-pod-1",
			Deployment:  "api",
			CPUUsage:    500,
			MemoryUsage: 256 * 1024 * 1024,
			CollectedAt: day,
		},
		{
			Namespace:   "default",
			PodName:     "worker-pod-1",
			Deployment:  "worker",
			CPUUsage:    300,
			MemoryUsage: 128 * 1024 * 1024,
			CollectedAt: day,
		},
	}
}

// intSliceEqual checks if two int slices are identical.
func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
