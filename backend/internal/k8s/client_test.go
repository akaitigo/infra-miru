package k8s_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

func newFakePod(namespace, name, deploymentName, cpuReq, cpuLim, memReq, memLim string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuReq),
							corev1.ResourceMemory: resource.MustParse(memReq),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuLim),
							corev1.ResourceMemory: resource.MustParse(memLim),
						},
					},
				},
			},
		},
	}

	if deploymentName != "" {
		pod.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "ReplicaSet",
				Name: deploymentName + "-abc123",
			},
		}
	}

	return pod
}

func TestListPods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pods      []*corev1.Pod
		opts      k8s.ListOptions
		wantCount int
		wantErr   bool
		checkFunc func(t *testing.T, pods []k8s.PodResource)
	}{
		{
			name: "list all pods across namespaces",
			pods: []*corev1.Pod{
				newFakePod("default", "pod-1", "api", "100m", "200m", "128Mi", "256Mi"),
				newFakePod("kube-system", "pod-2", "dns", "50m", "100m", "64Mi", "128Mi"),
			},
			opts:      k8s.ListOptions{},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "filter by namespace",
			pods: []*corev1.Pod{
				newFakePod("default", "pod-1", "api", "100m", "200m", "128Mi", "256Mi"),
				newFakePod("kube-system", "pod-2", "dns", "50m", "100m", "64Mi", "128Mi"),
			},
			opts:      k8s.ListOptions{Namespace: "default"},
			wantCount: 1,
			wantErr:   false,
			checkFunc: func(t *testing.T, pods []k8s.PodResource) {
				t.Helper()
				if pods[0].Namespace != "default" {
					t.Errorf("Namespace = %q, want %q", pods[0].Namespace, "default")
				}
			},
		},
		{
			name: "filter by deployment",
			pods: []*corev1.Pod{
				newFakePod("default", "api-abc123-xyz", "api", "100m", "200m", "128Mi", "256Mi"),
				newFakePod("default", "worker-def456-uvw", "worker", "200m", "400m", "256Mi", "512Mi"),
			},
			opts:      k8s.ListOptions{Namespace: "default", Deployment: "api"},
			wantCount: 1,
			wantErr:   false,
			checkFunc: func(t *testing.T, pods []k8s.PodResource) {
				t.Helper()
				if pods[0].Deployment != "api" {
					t.Errorf("Deployment = %q, want %q", pods[0].Deployment, "api")
				}
			},
		},
		{
			name:      "empty cluster returns empty list",
			pods:      []*corev1.Pod{},
			opts:      k8s.ListOptions{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "resource values are correct",
			pods: []*corev1.Pod{
				newFakePod("default", "pod-1", "api", "250m", "500m", "256Mi", "512Mi"),
			},
			opts:      k8s.ListOptions{},
			wantCount: 1,
			wantErr:   false,
			checkFunc: func(t *testing.T, pods []k8s.PodResource) {
				t.Helper()
				p := pods[0]
				if p.CPURequest != 250 {
					t.Errorf("CPURequest = %d, want 250 millicores", p.CPURequest)
				}
				if p.CPULimit != 500 {
					t.Errorf("CPULimit = %d, want 500 millicores", p.CPULimit)
				}
				// 256Mi = 256 * 1024 * 1024 = 268435456 bytes
				if p.MemoryRequest != 268435456 {
					t.Errorf("MemoryRequest = %d, want 268435456 bytes", p.MemoryRequest)
				}
				// 512Mi = 512 * 1024 * 1024 = 536870912 bytes
				if p.MemoryLimit != 536870912 {
					t.Errorf("MemoryLimit = %d, want 536870912 bytes", p.MemoryLimit)
				}
			},
		},
		{
			name: "multi-container pod aggregates resources",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-container",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "app",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("128Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("200m"),
										corev1.ResourceMemory: resource.MustParse("256Mi"),
									},
								},
							},
							{
								Name: "sidecar",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("50m"),
										corev1.ResourceMemory: resource.MustParse("64Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("128Mi"),
									},
								},
							},
						},
					},
				},
			},
			opts:      k8s.ListOptions{},
			wantCount: 1,
			wantErr:   false,
			checkFunc: func(t *testing.T, pods []k8s.PodResource) {
				t.Helper()
				p := pods[0]
				if p.CPURequest != 150 {
					t.Errorf("CPURequest = %d, want 150 millicores (100+50)", p.CPURequest)
				}
				if p.CPULimit != 300 {
					t.Errorf("CPULimit = %d, want 300 millicores (200+100)", p.CPULimit)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			objects := make([]corev1.Pod, len(tt.pods))
			for i, p := range tt.pods {
				objects[i] = *p
			}

			fakeCS := fake.NewClientset()
			for i := range tt.pods {
				_, err := fakeCS.CoreV1().Pods(tt.pods[i].Namespace).Create(
					context.Background(), tt.pods[i], metav1.CreateOptions{},
				)
				if err != nil {
					t.Fatalf("failed to create fake pod: %v", err)
				}
			}

			// nil metricsClient is safe; fetchPodMetrics returns empty map.
			client := k8s.NewClient(fakeCS, nil)
			pods, err := client.ListPods(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(pods) != tt.wantCount {
				t.Errorf("got %d pods, want %d", len(pods), tt.wantCount)
			}

			if tt.checkFunc != nil && len(pods) > 0 {
				tt.checkFunc(t, pods)
			}
		})
	}
}
