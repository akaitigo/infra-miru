// Package cost provides resource cost calculation and optimization recommendations
// for Kubernetes workloads based on their actual resource utilization.
package cost

import (
	"fmt"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
)

const (
	// DefaultCPUPricePerMillicorePerMonth is the GCP e2-standard series baseline
	// price in JPY for 1 millicore per month.
	DefaultCPUPricePerMillicorePerMonth float64 = 3.5

	// DefaultMemoryPricePerMBPerMonth is the GCP e2-standard series baseline
	// price in JPY for 1 MB of memory per month.
	DefaultMemoryPricePerMBPerMonth float64 = 0.5

	// bytesPerMB converts bytes to megabytes.
	bytesPerMB int64 = 1024 * 1024
)

// Recommendation represents a cost optimization suggestion for a Deployment.
type Recommendation struct {
	Namespace            string `json:"namespace"`
	Deployment           string `json:"deployment"`
	Message              string `json:"message"`
	CurrentRequestCPU    int64  `json:"current_request_cpu_millicores"`
	CurrentRequestMemory int64  `json:"current_request_memory_bytes"`
	RecommendedCPU       int64  `json:"recommended_cpu_millicores"`
	RecommendedMemory    int64  `json:"recommended_memory_bytes"`
	MonthlySavingsJPY    int64  `json:"monthly_savings_jpy"`
}

// Calculator computes cost optimization recommendations based on resource analysis.
type Calculator struct {
	cpuPricePerMillicorePerMonth float64
	memoryPricePerMBPerMonth     float64
}

// NewCalculator creates a Calculator with the default GCP-based pricing.
func NewCalculator() *Calculator {
	return &Calculator{
		cpuPricePerMillicorePerMonth: DefaultCPUPricePerMillicorePerMonth,
		memoryPricePerMBPerMonth:     DefaultMemoryPricePerMBPerMonth,
	}
}

// NewCalculatorWithPricing creates a Calculator with custom per-unit pricing.
func NewCalculatorWithPricing(cpuPrice, memoryPrice float64) *Calculator {
	return &Calculator{
		cpuPricePerMillicorePerMonth: cpuPrice,
		memoryPricePerMBPerMonth:     memoryPrice,
	}
}

// CalculateRecommendations generates cost optimization recommendations for
// over-provisioned resources. Only Pods with divergence >= 50% are included.
// Recommendations are aggregated by Deployment.
func (c *Calculator) CalculateRecommendations(analyses []analyzer.ResourceAnalysis) []Recommendation {
	// Aggregate by namespace/deployment.
	type deploymentAccum struct { //nolint:govet // fieldalignment: local accumulator struct, readability over alignment
		namespace       string
		deployment      string
		cpuDivergences  []float64
		memDivergences  []float64
		totalCPURequest int64
		totalCPUUsage   int64
		totalMemRequest int64
		totalMemUsage   int64
	}

	var keys []string
	index := make(map[string]int)
	var accums []deploymentAccum

	for i := range analyses {
		a := &analyses[i]
		if !a.IsOverProvisioned {
			continue
		}

		key := a.Namespace + "/" + a.Deployment

		idx, exists := index[key]
		if !exists {
			idx = len(accums)
			index[key] = idx
			keys = append(keys, key)
			accums = append(accums, deploymentAccum{
				namespace:  a.Namespace,
				deployment: a.Deployment,
			})
		}

		acc := &accums[idx]
		acc.totalCPURequest += a.CPURequest
		acc.totalCPUUsage += a.CPUUsage
		acc.totalMemRequest += a.MemoryRequest
		acc.totalMemUsage += a.MemoryUsage
		acc.cpuDivergences = append(acc.cpuDivergences, a.CPUDivergence)
		acc.memDivergences = append(acc.memDivergences, a.MemoryDivergence)
	}

	recommendations := make([]Recommendation, 0, len(keys))

	for _, key := range keys {
		idx := index[key]
		acc := &accums[idx]

		avgCPUDiv := avg(acc.cpuDivergences)
		avgMemDiv := avg(acc.memDivergences)

		// Recommended = Usage * 1.2 (20% buffer for safety).
		// Ensure recommended is at least 1 if there's any usage.
		recommendedCPU := computeRecommended(acc.totalCPUUsage)
		recommendedMem := computeRecommended(acc.totalMemUsage)

		cpuSavings := c.cpuSavings(acc.totalCPURequest, recommendedCPU)
		memSavings := c.memorySavings(acc.totalMemRequest, recommendedMem)
		totalSavings := int64(cpuSavings + memSavings)

		maxDiv := avgCPUDiv
		if avgMemDiv > maxDiv {
			maxDiv = avgMemDiv
		}

		reductionPercent := int64(maxDiv / 2)
		if reductionPercent < 1 {
			reductionPercent = 1
		}

		message := fmt.Sprintf(
			"このDeploymentはRequestの%.0f%%しか使っていない → Requestを%d%%削減で月¥%d節約",
			100-maxDiv, reductionPercent, totalSavings,
		)

		recommendations = append(recommendations, Recommendation{
			Namespace:            acc.namespace,
			Deployment:           acc.deployment,
			CurrentRequestCPU:    acc.totalCPURequest,
			CurrentRequestMemory: acc.totalMemRequest,
			RecommendedCPU:       recommendedCPU,
			RecommendedMemory:    recommendedMem,
			MonthlySavingsJPY:    totalSavings,
			Message:              message,
		})
	}

	return recommendations
}

// computeRecommended returns the recommended resource amount: usage * 1.2 (20% buffer).
// If usage is 0, returns 0.
func computeRecommended(usage int64) int64 {
	if usage == 0 {
		return 0
	}
	recommended := int64(float64(usage) * 1.2)
	if recommended < 1 {
		recommended = 1
	}
	return recommended
}

// cpuSavings calculates monthly CPU cost savings in JPY.
func (c *Calculator) cpuSavings(currentRequest, recommended int64) float64 {
	diff := currentRequest - recommended
	if diff <= 0 {
		return 0
	}
	return float64(diff) * c.cpuPricePerMillicorePerMonth
}

// memorySavings calculates monthly memory cost savings in JPY.
func (c *Calculator) memorySavings(currentRequest, recommended int64) float64 {
	diffBytes := currentRequest - recommended
	if diffBytes <= 0 {
		return 0
	}
	diffMB := float64(diffBytes) / float64(bytesPerMB)
	return diffMB * c.memoryPricePerMBPerMonth
}

// avg computes the arithmetic mean of a float64 slice.
func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
