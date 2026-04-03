package analyzer

import (
	"bytes"
	"fmt"
	"text/template"
)

// CronHPAConfig holds the configuration for generating a CronHPA template.
type CronHPAConfig struct {
	Namespace     string `json:"namespace"`
	Deployment    string `json:"deployment"`
	ScaleDownTime string `json:"scale_down_time"` // e.g. "22:00" JST
	ScaleUpTime   string `json:"scale_up_time"`   // e.g. "06:00" JST
	MinReplicas   int32  `json:"min_replicas"`
	MaxReplicas   int32  `json:"max_replicas"`
}

// cronHPATemplate is the Kubernetes CronJob + kubectl patch YAML template.
// This pattern is more portable than vendor-specific CronHPA CRDs.
const cronHPATemplate = `---
# Scale-down CronJob: reduces replicas during low-load hours
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{ .Deployment }}-scale-down
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/managed-by: infra-miru
    app.kubernetes.io/part-of: {{ .Deployment }}
spec:
  schedule: "{{ .ScaleDownCron }}"
  timeZone: "Asia/Tokyo"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: {{ .Deployment }}-scaler
          restartPolicy: OnFailure
          containers:
            - name: scale-down
              image: bitnami/kubectl:latest
              command:
                - /bin/sh
                - -c
                - |
                  kubectl patch hpa {{ .Deployment }} \
                    -n {{ .Namespace }} \
                    -p '{"spec":{"minReplicas":{{ .MinReplicas }},"maxReplicas":{{ .MaxReplicas }}}}'
---
# Scale-up CronJob: restores replicas for normal load hours
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{ .Deployment }}-scale-up
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/managed-by: infra-miru
    app.kubernetes.io/part-of: {{ .Deployment }}
spec:
  schedule: "{{ .ScaleUpCron }}"
  timeZone: "Asia/Tokyo"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: {{ .Deployment }}-scaler
          restartPolicy: OnFailure
          containers:
            - name: scale-up
              image: bitnami/kubectl:latest
              command:
                - /bin/sh
                - -c
                - |
                  kubectl patch hpa {{ .Deployment }} \
                    -n {{ .Namespace }} \
                    -p '{"spec":{"minReplicas":{{ .NormalMinReplicas }},"maxReplicas":{{ .NormalMaxReplicas }}}}'
---
# ServiceAccount for the scaler CronJobs
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Deployment }}-scaler
  namespace: {{ .Namespace }}
---
# Role granting permission to patch HPA
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ .Deployment }}-hpa-patcher
  namespace: {{ .Namespace }}
rules:
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    resourceNames: ["{{ .Deployment }}"]
    verbs: ["get", "patch"]
---
# RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ .Deployment }}-hpa-patcher
  namespace: {{ .Namespace }}
subjects:
  - kind: ServiceAccount
    name: {{ .Deployment }}-scaler
    namespace: {{ .Namespace }}
roleRef:
  kind: Role
  name: {{ .Deployment }}-hpa-patcher
  apiGroup: rbac.authorization.k8s.io
`

// templateData holds the expanded values for YAML rendering.
type templateData struct {
	Namespace         string
	Deployment        string
	ScaleDownCron     string
	ScaleUpCron       string
	MinReplicas       int32
	MaxReplicas       int32
	NormalMinReplicas int32
	NormalMaxReplicas int32
}

// GenerateCronHPA creates a CronHPAConfig from a ScheduleAnalysis.
// It picks the earliest and latest low-load hours as scale-down/up boundaries.
// Default replica counts: low-load min=1, max=2; normal min=2, max=10.
func GenerateCronHPA(schedule ScheduleAnalysis) CronHPAConfig {
	scaleDown := "22:00"
	scaleUp := "06:00"

	if len(schedule.LowLoadHours) > 0 {
		scaleDown = fmt.Sprintf("%02d:00", schedule.LowLoadHours[0])

		lastLow := schedule.LowLoadHours[len(schedule.LowLoadHours)-1]
		scaleUpHour := (lastLow + 1) % 24
		scaleUp = fmt.Sprintf("%02d:00", scaleUpHour)
	}

	return CronHPAConfig{
		Namespace:     schedule.Namespace,
		Deployment:    schedule.Deployment,
		ScaleDownTime: scaleDown,
		ScaleUpTime:   scaleUp,
		MinReplicas:   1,
		MaxReplicas:   2,
	}
}

// RenderCronHPAYAML renders the CronHPA configuration as Kubernetes YAML manifests.
// The generated YAML includes CronJobs for scale-down and scale-up, plus the
// required ServiceAccount, Role, and RoleBinding.
func RenderCronHPAYAML(config CronHPAConfig) (string, error) {
	tmpl, err := template.New("cronhpa").Parse(cronHPATemplate)
	if err != nil {
		return "", fmt.Errorf("parse cronhpa template: %w", err)
	}

	scaleDownCron, err := timeToCron(config.ScaleDownTime)
	if err != nil {
		return "", fmt.Errorf("invalid scale_down_time: %w", err)
	}

	scaleUpCron, err := timeToCron(config.ScaleUpTime)
	if err != nil {
		return "", fmt.Errorf("invalid scale_up_time: %w", err)
	}

	data := templateData{
		Namespace:         config.Namespace,
		Deployment:        config.Deployment,
		ScaleDownCron:     scaleDownCron,
		ScaleUpCron:       scaleUpCron,
		MinReplicas:       config.MinReplicas,
		MaxReplicas:       config.MaxReplicas,
		NormalMinReplicas: config.MinReplicas * 2,
		NormalMaxReplicas: config.MaxReplicas * 5,
	}

	if data.NormalMinReplicas < 2 {
		data.NormalMinReplicas = 2
	}
	if data.NormalMaxReplicas < 10 {
		data.NormalMaxReplicas = 10
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute cronhpa template: %w", err)
	}

	return buf.String(), nil
}

// timeToCron converts a "HH:MM" time string to a cron schedule expression.
// Example: "22:00" -> "0 22 * * *".
func timeToCron(timeStr string) (string, error) {
	var hour, minute int

	n, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return "", fmt.Errorf("parse time %q: %w", timeStr, err)
	}
	if n != 2 {
		return "", fmt.Errorf("parse time %q: expected HH:MM format", timeStr)
	}
	if hour < 0 || hour > 23 {
		return "", fmt.Errorf("hour %d out of range [0, 23]", hour)
	}
	if minute < 0 || minute > 59 {
		return "", fmt.Errorf("minute %d out of range [0, 59]", minute)
	}

	return fmt.Sprintf("%d %d * * *", minute, hour), nil
}
