package analyzer_test

import (
	"strings"
	"testing"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
)

func TestGenerateCronHPA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		schedule        analyzer.ScheduleAnalysis
		wantScaleDown   string
		wantScaleUp     string
		wantMinReplicas int32
		wantMaxReplicas int32
	}{
		{
			name: "uses low load hours for scale-down/up times",
			schedule: analyzer.ScheduleAnalysis{
				Namespace:    "prod",
				Deployment:   "api",
				LowLoadHours: []int{0, 1, 2, 3, 4, 5},
			},
			wantScaleDown:   "00:00",
			wantScaleUp:     "06:00",
			wantMinReplicas: 1,
			wantMaxReplicas: 2,
		},
		{
			name: "single low load hour",
			schedule: analyzer.ScheduleAnalysis{
				Namespace:    "staging",
				Deployment:   "worker",
				LowLoadHours: []int{3},
			},
			wantScaleDown:   "03:00",
			wantScaleUp:     "04:00",
			wantMinReplicas: 1,
			wantMaxReplicas: 2,
		},
		{
			name: "no low load hours uses defaults",
			schedule: analyzer.ScheduleAnalysis{
				Namespace:    "default",
				Deployment:   "web",
				LowLoadHours: nil,
			},
			wantScaleDown:   "22:00",
			wantScaleUp:     "06:00",
			wantMinReplicas: 1,
			wantMaxReplicas: 2,
		},
		{
			name: "low load hour at 23 wraps scale-up to 0",
			schedule: analyzer.ScheduleAnalysis{
				Namespace:    "default",
				Deployment:   "batch",
				LowLoadHours: []int{22, 23},
			},
			wantScaleDown:   "22:00",
			wantScaleUp:     "00:00",
			wantMinReplicas: 1,
			wantMaxReplicas: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := analyzer.GenerateCronHPA(tt.schedule)

			if config.Namespace != tt.schedule.Namespace {
				t.Errorf("Namespace = %q, want %q", config.Namespace, tt.schedule.Namespace)
			}
			if config.Deployment != tt.schedule.Deployment {
				t.Errorf("Deployment = %q, want %q", config.Deployment, tt.schedule.Deployment)
			}
			if config.ScaleDownTime != tt.wantScaleDown {
				t.Errorf("ScaleDownTime = %q, want %q", config.ScaleDownTime, tt.wantScaleDown)
			}
			if config.ScaleUpTime != tt.wantScaleUp {
				t.Errorf("ScaleUpTime = %q, want %q", config.ScaleUpTime, tt.wantScaleUp)
			}
			if config.MinReplicas != tt.wantMinReplicas {
				t.Errorf("MinReplicas = %d, want %d", config.MinReplicas, tt.wantMinReplicas)
			}
			if config.MaxReplicas != tt.wantMaxReplicas {
				t.Errorf("MaxReplicas = %d, want %d", config.MaxReplicas, tt.wantMaxReplicas)
			}
		})
	}
}

func TestRenderCronHPAYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   analyzer.CronHPAConfig
		wantErr  bool
		contains []string
	}{
		{
			name: "valid config produces valid YAML",
			config: analyzer.CronHPAConfig{
				Namespace:     "prod",
				Deployment:    "api",
				ScaleDownTime: "22:00",
				ScaleUpTime:   "06:00",
				MinReplicas:   1,
				MaxReplicas:   2,
			},
			wantErr: false,
			contains: []string{
				"kind: CronJob",
				"api-scale-down",
				"api-scale-up",
				"namespace: prod",
				"0 22 * * *",
				"0 6 * * *",
				"api-scaler",
				"api-hpa-patcher",
				"kind: ServiceAccount",
				"kind: Role",
				"kind: RoleBinding",
			},
		},
		{
			name: "different deployment name appears in YAML",
			config: analyzer.CronHPAConfig{
				Namespace:     "staging",
				Deployment:    "worker",
				ScaleDownTime: "00:00",
				ScaleUpTime:   "06:00",
				MinReplicas:   1,
				MaxReplicas:   3,
			},
			wantErr: false,
			contains: []string{
				"worker-scale-down",
				"worker-scale-up",
				"namespace: staging",
				"worker-scaler",
			},
		},
		{
			name: "invalid time format returns error",
			config: analyzer.CronHPAConfig{
				Namespace:     "default",
				Deployment:    "test",
				ScaleDownTime: "invalid",
				ScaleUpTime:   "06:00",
				MinReplicas:   1,
				MaxReplicas:   2,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			yaml, err := analyzer.RenderCronHPAYAML(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if yaml == "" {
				t.Fatal("expected non-empty YAML")
			}

			for _, s := range tt.contains {
				if !strings.Contains(yaml, s) {
					t.Errorf("YAML does not contain %q", s)
				}
			}
		})
	}
}

func TestRenderCronHPAYAMLNormalReplicas(t *testing.T) {
	t.Parallel()

	config := analyzer.CronHPAConfig{
		Namespace:     "prod",
		Deployment:    "api",
		ScaleDownTime: "22:00",
		ScaleUpTime:   "06:00",
		MinReplicas:   1,
		MaxReplicas:   2,
	}

	yaml, err := analyzer.RenderCronHPAYAML(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// NormalMinReplicas = max(MinReplicas*2, 2) = 2
	// NormalMaxReplicas = max(MaxReplicas*5, 10) = 10
	if !strings.Contains(yaml, `"minReplicas":2`) {
		t.Error("expected normal minReplicas of 2 in scale-up command")
	}
	if !strings.Contains(yaml, `"maxReplicas":10`) {
		t.Error("expected normal maxReplicas of 10 in scale-up command")
	}
}
