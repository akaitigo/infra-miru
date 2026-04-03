package analyzer

import (
	"sort"
	"time"

	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// jst is the Asia/Tokyo timezone used for schedule analysis.
var jst = time.FixedZone("JST", 9*60*60)

// lowLoadThresholdRatio defines the ratio below the overall average that qualifies
// as "low load". An hour is low-load if its average CPU usage is below
// overallAverage * lowLoadThresholdRatio.
const lowLoadThresholdRatio = 0.30

// HourlyLoad holds aggregated resource usage for a specific hour of the day.
type HourlyLoad struct {
	Hour           int     `json:"hour"`
	AvgCPUUsage    float64 `json:"avg_cpu_usage_millicores"`
	AvgMemoryUsage float64 `json:"avg_memory_usage_bytes"`
	SampleCount    int     `json:"sample_count"`
}

// ScheduleAnalysis holds time-based load analysis for a single Deployment.
type ScheduleAnalysis struct {
	Namespace        string       `json:"namespace"`
	Deployment       string       `json:"deployment"`
	HourlyLoads      []HourlyLoad `json:"hourly_loads"`
	LowLoadHours     []int        `json:"low_load_hours"`
	IsWeekendLowLoad bool         `json:"is_weekend_low_load"`
}

// hourAccum accumulates CPU and memory usage for a specific hour.
type hourAccum struct {
	cpuSum float64
	memSum float64
	count  int
}

// deployAccum accumulates schedule data for a single deployment.
type deployAccum struct {
	hours        [24]hourAccum
	weekdayCPU   float64
	weekdayCount int
	weekendCPU   float64
	weekendCount int
}

// deployKey uniquely identifies a deployment within a namespace.
type deployKey struct {
	namespace  string
	deployment string
}

// AnalyzeSchedule groups Pods by Deployment and computes hourly load patterns.
// It detects low-load hours (below 30% of overall average CPU) and weekend patterns.
func (a *Analyzer) AnalyzeSchedule(pods []k8s.PodResource) []ScheduleAnalysis {
	if len(pods) == 0 {
		return []ScheduleAnalysis{}
	}

	// Preserve insertion order.
	var keys []deployKey
	index := make(map[deployKey]int)
	var accums []deployAccum

	for i := range pods {
		pod := &pods[i]
		if pod.Deployment == "" {
			continue
		}

		key := deployKey{namespace: pod.Namespace, deployment: pod.Deployment}

		idx, exists := index[key]
		if !exists {
			idx = len(accums)
			index[key] = idx
			keys = append(keys, key)
			accums = append(accums, deployAccum{})
		}

		acc := &accums[idx]

		jstTime := pod.CollectedAt.In(jst)
		hour := jstTime.Hour()

		acc.hours[hour].cpuSum += float64(pod.CPUUsage)
		acc.hours[hour].memSum += float64(pod.MemoryUsage)
		acc.hours[hour].count++

		weekday := jstTime.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			acc.weekendCPU += float64(pod.CPUUsage)
			acc.weekendCount++
		} else {
			acc.weekdayCPU += float64(pod.CPUUsage)
			acc.weekdayCount++
		}
	}

	results := make([]ScheduleAnalysis, 0, len(keys))

	for _, key := range keys {
		idx := index[key]
		acc := &accums[idx]

		hourlyLoads := buildHourlyLoads(&acc.hours)
		overallAvg := overallAverageCPU(hourlyLoads)
		lowLoadHours := detectLowLoadHours(hourlyLoads, overallAvg)
		weekendLow := isWeekendLowLoad(acc)

		results = append(results, ScheduleAnalysis{
			Namespace:        key.namespace,
			Deployment:       key.deployment,
			HourlyLoads:      hourlyLoads,
			LowLoadHours:     lowLoadHours,
			IsWeekendLowLoad: weekendLow,
		})
	}

	return results
}

// buildHourlyLoads converts the 24-hour accumulator array into a slice of HourlyLoad.
// Only hours with at least one sample are included.
func buildHourlyLoads(hours *[24]hourAccum) []HourlyLoad {
	loads := make([]HourlyLoad, 0, 24)

	for h := 0; h < 24; h++ {
		if hours[h].count == 0 {
			continue
		}
		loads = append(loads, HourlyLoad{
			Hour:           h,
			AvgCPUUsage:    hours[h].cpuSum / float64(hours[h].count),
			AvgMemoryUsage: hours[h].memSum / float64(hours[h].count),
			SampleCount:    hours[h].count,
		})
	}

	return loads
}

// overallAverageCPU computes the average CPU usage across all hourly loads,
// weighted by sample count.
func overallAverageCPU(loads []HourlyLoad) float64 {
	if len(loads) == 0 {
		return 0
	}

	totalCPU := 0.0
	totalSamples := 0

	for i := range loads {
		totalCPU += loads[i].AvgCPUUsage * float64(loads[i].SampleCount)
		totalSamples += loads[i].SampleCount
	}

	if totalSamples == 0 {
		return 0
	}

	return totalCPU / float64(totalSamples)
}

// detectLowLoadHours returns hours where the average CPU usage is below the
// threshold (overallAvg * lowLoadThresholdRatio). Results are sorted ascending.
func detectLowLoadHours(loads []HourlyLoad, overallAvg float64) []int {
	threshold := overallAvg * lowLoadThresholdRatio

	var hours []int
	for i := range loads {
		if loads[i].AvgCPUUsage < threshold {
			hours = append(hours, loads[i].Hour)
		}
	}

	sort.Ints(hours)

	return hours
}

// isWeekendLowLoad determines whether the weekend average CPU is significantly
// lower than the weekday average. It uses the same 30% threshold ratio.
func isWeekendLowLoad(acc *deployAccum) bool {
	if acc.weekdayCount == 0 || acc.weekendCount == 0 {
		return false
	}

	weekdayAvg := acc.weekdayCPU / float64(acc.weekdayCount)
	weekendAvg := acc.weekendCPU / float64(acc.weekendCount)

	if weekdayAvg == 0 {
		return false
	}

	return weekendAvg < weekdayAvg*lowLoadThresholdRatio
}
