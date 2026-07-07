package k8s

import (
	"errors"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// inClusterConfigFunc loads a REST config from the in-cluster environment.
type inClusterConfigFunc func() (*rest.Config, error)

// fromFlagsConfigFunc loads a REST config from a kubeconfig file.
type fromFlagsConfigFunc func(masterURL, kubeconfigPath string) (*rest.Config, error)

// Connect creates a Client, preferring the in-cluster configuration (available
// when running inside a Kubernetes Pod) and falling back to the kubeconfig file
// at the given path when the process is not running inside a cluster.
func Connect(kubeconfigPath string) (*Client, error) {
	config, err := resolveConfig(kubeconfigPath, rest.InClusterConfig, clientcmd.BuildConfigFromFlags)
	if err != nil {
		return nil, err
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

// resolveConfig resolves a Kubernetes REST config. It first attempts the
// in-cluster configuration and falls back to the kubeconfig file when the
// process is not running inside a cluster. When both sources fail the returned
// error wraps both underlying failures.
func resolveConfig(
	kubeconfigPath string,
	inCluster inClusterConfigFunc,
	fromFlags fromFlagsConfigFunc,
) (*rest.Config, error) {
	config, inClusterErr := inCluster()
	if inClusterErr == nil {
		return config, nil
	}

	config, flagsErr := fromFlags("", kubeconfigPath)
	if flagsErr != nil {
		return nil, fmt.Errorf(
			"resolve kubernetes config (kubeconfig %q): %w",
			kubeconfigPath,
			errors.Join(inClusterErr, flagsErr),
		)
	}

	return config, nil
}
