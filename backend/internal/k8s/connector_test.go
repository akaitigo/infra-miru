package k8s

import (
	"errors"
	"testing"

	"k8s.io/client-go/rest"
)

func TestResolveConfig(t *testing.T) {
	t.Parallel()

	const (
		inClusterHost  = "https://in-cluster.local:443"
		kubeconfigHost = "https://kubeconfig.local:6443"
	)

	errKubeconfigMissing := errors.New("kubeconfig not found")

	tests := []struct {
		name      string
		inCluster inClusterConfigFunc
		fromFlags fromFlagsConfigFunc
		wantHost  string
		wantErr   bool
		wantErrIs []error
	}{
		{
			name: "prefers in-cluster config when running inside a cluster",
			inCluster: func() (*rest.Config, error) {
				return &rest.Config{Host: inClusterHost}, nil
			},
			fromFlags: func(_, _ string) (*rest.Config, error) {
				return &rest.Config{Host: kubeconfigHost}, nil
			},
			wantHost: inClusterHost,
		},
		{
			name: "falls back to kubeconfig when not running in a cluster",
			inCluster: func() (*rest.Config, error) {
				return nil, rest.ErrNotInCluster
			},
			fromFlags: func(_, _ string) (*rest.Config, error) {
				return &rest.Config{Host: kubeconfigHost}, nil
			},
			wantHost: kubeconfigHost,
		},
		{
			name: "returns error wrapping both failures when in-cluster and kubeconfig fail",
			inCluster: func() (*rest.Config, error) {
				return nil, rest.ErrNotInCluster
			},
			fromFlags: func(_, _ string) (*rest.Config, error) {
				return nil, errKubeconfigMissing
			},
			wantErr:   true,
			wantErrIs: []error{rest.ErrNotInCluster, errKubeconfigMissing},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := resolveConfig("/fake/kubeconfig", tt.inCluster, tt.fromFlags)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				for _, target := range tt.wantErrIs {
					if !errors.Is(err, target) {
						t.Errorf("error %q does not wrap %q", err, target)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config == nil {
				t.Fatal("expected config, got nil")
			}
			if config.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", config.Host, tt.wantHost)
			}
		})
	}
}
