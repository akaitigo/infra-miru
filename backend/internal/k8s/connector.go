package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Connect creates a Client from the kubeconfig file at the given path.
func Connect(kubeconfigPath string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create metrics clientset: %w", err)
	}

	return NewClient(clientset, metricsClient), nil
}
