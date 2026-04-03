package k8s

import "time"

// PodResource represents a Kubernetes Pod with its resource requests, limits, and actual usage.
type PodResource struct {
	CollectedAt   time.Time
	Namespace     string
	PodName       string
	Deployment    string
	CPURequest    int64 // millicores
	CPULimit      int64 // millicores
	CPUUsage      int64 // millicores
	MemoryRequest int64 // bytes
	MemoryLimit   int64 // bytes
	MemoryUsage   int64 // bytes
}

// ListOptions specifies filter criteria for listing Pods.
type ListOptions struct {
	Namespace  string
	Deployment string
}
