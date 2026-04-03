// Package analyzer provides resource utilization analysis for Kubernetes Pods.
// It calculates divergence between requested and actual resource usage, and
// aggregates results by Deployment.
package analyzer

import (
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// ResourceAnalysis holds the result of analyzing a single Pod's resource utilization.
type ResourceAnalysis struct {
	Namespace         string  `json:"namespace"`
	PodName           string  `json:"pod_name"`
	Deployment        string  `json:"deployment"`
	CPURequest        int64   `json:"cpu_request_millicores"`
	CPUUsage          int64   `json:"cpu_usage_millicores"`
	CPUDivergence     float64 `json:"cpu_divergence_percent"`
	MemoryRequest     int64   `json:"memory_request_bytes"`
	MemoryUsage       int64   `json:"memory_usage_bytes"`
	MemoryDivergence  float64 `json:"memory_divergence_percent"`
	IsOverProvisioned bool    `json:"is_over_provisioned"`
}

// DeploymentSummary aggregates resource analysis results for all Pods in a Deployment.
type DeploymentSummary struct {
	Namespace           string  `json:"namespace"`
	Deployment          string  `json:"deployment"`
	PodCount            int     `json:"pod_count"`
	TotalCPURequest     int64   `json:"total_cpu_request_millicores"`
	TotalCPUUsage       int64   `json:"total_cpu_usage_millicores"`
	AvgCPUDivergence    float64 `json:"avg_cpu_divergence_percent"`
	TotalMemoryRequest  int64   `json:"total_memory_request_bytes"`
	TotalMemoryUsage    int64   `json:"total_memory_usage_bytes"`
	AvgMemoryDivergence float64 `json:"avg_memory_divergence_percent"`
	IsOverProvisioned   bool    `json:"is_over_provisioned"`
}

// Analyzer performs resource utilization analysis on Pod data.
type Analyzer struct{}

// NewAnalyzer creates a new Analyzer instance.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzeResources computes resource divergence for each Pod.
// Divergence = ((Request - Usage) / Request) * 100.
// If Request is 0, divergence is reported as 0 (no over-provisioning possible).
// A Pod is considered over-provisioned if either CPU or Memory divergence >= 50%.
func (a *Analyzer) AnalyzeResources(pods []k8s.PodResource) []ResourceAnalysis {
	results := make([]ResourceAnalysis, 0, len(pods))

	for i := range pods {
		pod := &pods[i]

		cpuDiv := calcDivergence(pod.CPURequest, pod.CPUUsage)
		memDiv := calcDivergence(pod.MemoryRequest, pod.MemoryUsage)

		analysis := ResourceAnalysis{
			Namespace:         pod.Namespace,
			PodName:           pod.PodName,
			Deployment:        pod.Deployment,
			CPURequest:        pod.CPURequest,
			CPUUsage:          pod.CPUUsage,
			CPUDivergence:     cpuDiv,
			MemoryRequest:     pod.MemoryRequest,
			MemoryUsage:       pod.MemoryUsage,
			MemoryDivergence:  memDiv,
			IsOverProvisioned: cpuDiv >= 50 || memDiv >= 50,
		}

		results = append(results, analysis)
	}

	return results
}

// AggregateByDeployment groups ResourceAnalysis results by Deployment and computes
// aggregate metrics (totals and averages).
func (a *Analyzer) AggregateByDeployment(analyses []ResourceAnalysis) []DeploymentSummary {
	type accumulator struct {
		namespace     string
		podCount      int
		totalCPUReq   int64
		totalCPUUsage int64
		totalCPUDiv   float64
		totalMemReq   int64
		totalMemUsage int64
		totalMemDiv   float64
	}

	// Use a slice to preserve insertion order and an index map for lookups.
	var keys []string
	index := make(map[string]int)
	var accums []accumulator

	for i := range analyses {
		ra := &analyses[i]
		key := ra.Namespace + "/" + ra.Deployment

		idx, exists := index[key]
		if !exists {
			idx = len(accums)
			index[key] = idx
			keys = append(keys, key)
			accums = append(accums, accumulator{
				namespace: ra.Namespace,
			})
		}

		acc := &accums[idx]
		acc.podCount++
		acc.totalCPUReq += ra.CPURequest
		acc.totalCPUUsage += ra.CPUUsage
		acc.totalCPUDiv += ra.CPUDivergence
		acc.totalMemReq += ra.MemoryRequest
		acc.totalMemUsage += ra.MemoryUsage
		acc.totalMemDiv += ra.MemoryDivergence
	}

	summaries := make([]DeploymentSummary, 0, len(keys))

	for _, key := range keys {
		idx := index[key]
		acc := &accums[idx]

		// Find the deployment name from the key.
		// Key format is "namespace/deployment".
		deployment := ""
		for i := range analyses {
			if analyses[i].Namespace+"/"+analyses[i].Deployment == key {
				deployment = analyses[i].Deployment
				break
			}
		}

		avgCPUDiv := 0.0
		avgMemDiv := 0.0
		if acc.podCount > 0 {
			avgCPUDiv = acc.totalCPUDiv / float64(acc.podCount)
			avgMemDiv = acc.totalMemDiv / float64(acc.podCount)
		}

		summaries = append(summaries, DeploymentSummary{
			Namespace:           acc.namespace,
			Deployment:          deployment,
			PodCount:            acc.podCount,
			TotalCPURequest:     acc.totalCPUReq,
			TotalCPUUsage:       acc.totalCPUUsage,
			AvgCPUDivergence:    avgCPUDiv,
			TotalMemoryRequest:  acc.totalMemReq,
			TotalMemoryUsage:    acc.totalMemUsage,
			AvgMemoryDivergence: avgMemDiv,
			IsOverProvisioned:   avgCPUDiv >= 50 || avgMemDiv >= 50,
		})
	}

	return summaries
}

// calcDivergence computes the percentage divergence between request and actual usage.
// Returns 0 if request is 0 (cannot be over-provisioned with no request).
func calcDivergence(request, usage int64) float64 {
	if request == 0 {
		return 0
	}
	return float64(request-usage) / float64(request) * 100
}
