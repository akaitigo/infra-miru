// Package k8s provides Kubernetes cluster connectivity and resource data collection.
package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Client provides methods to query Kubernetes cluster resources.
type Client struct {
	clientset     kubernetes.Interface
	metricsClient metricsv.Interface
}

// NewClient creates a Client from the given Kubernetes and metrics clientsets.
func NewClient(cs kubernetes.Interface, mc metricsv.Interface) *Client {
	return &Client{
		clientset:     cs,
		metricsClient: mc,
	}
}

// ListPods retrieves Pods matching the given filter options and collects their
// resource requests, limits, and actual usage from the Metrics API.
func (c *Client) ListPods(ctx context.Context, opts ListOptions) ([]PodResource, error) {
	namespace := opts.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	// Fetch metrics for actual usage data.
	metricsMap, err := c.fetchPodMetrics(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("fetch pod metrics: %w", err)
	}

	now := time.Now().UTC()
	var results []PodResource

	for i := range podList.Items {
		pod := &podList.Items[i]

		deployment := extractDeploymentName(pod)
		if opts.Deployment != "" && deployment != opts.Deployment {
			continue
		}

		pr := PodResource{
			Namespace:   pod.Namespace,
			PodName:     pod.Name,
			Deployment:  deployment,
			CollectedAt: now,
		}

		for j := range pod.Spec.Containers {
			container := &pod.Spec.Containers[j]
			pr.CPURequest += container.Resources.Requests.Cpu().MilliValue()
			pr.CPULimit += container.Resources.Limits.Cpu().MilliValue()
			pr.MemoryRequest += container.Resources.Requests.Memory().Value()
			pr.MemoryLimit += container.Resources.Limits.Memory().Value()
		}

		// Merge actual usage from metrics.
		key := pod.Namespace + "/" + pod.Name
		if usage, ok := metricsMap[key]; ok {
			pr.CPUUsage = usage.cpuMillicores
			pr.MemoryUsage = usage.memoryBytes
		}

		results = append(results, pr)
	}

	return results, nil
}

type podUsage struct {
	cpuMillicores int64
	memoryBytes   int64
}

func (c *Client) fetchPodMetrics(ctx context.Context, namespace string) (map[string]podUsage, error) {
	result := make(map[string]podUsage)

	if c.metricsClient == nil {
		return result, nil
	}

	metricsList, err := c.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pod metrics: %w", err)
	}

	for i := range metricsList.Items {
		pm := &metricsList.Items[i]
		var cpu, mem int64
		for j := range pm.Containers {
			cpu += pm.Containers[j].Usage.Cpu().MilliValue()
			mem += pm.Containers[j].Usage.Memory().Value()
		}
		key := pm.Namespace + "/" + pm.Name
		result[key] = podUsage{cpuMillicores: cpu, memoryBytes: mem}
	}

	return result, nil
}

// extractDeploymentName derives the deployment name from a Pod's owner references
// and naming convention. Pods owned by a ReplicaSet typically follow the pattern
// <deployment>-<replicaset-hash>-<pod-hash>.
func extractDeploymentName(pod *corev1.Pod) string {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "ReplicaSet" {
			// ReplicaSet name format: <deployment>-<hash>
			parts := strings.Split(ref.Name, "-")
			if len(parts) > 1 {
				return strings.Join(parts[:len(parts)-1], "-")
			}
			return ref.Name
		}
	}
	return ""
}
