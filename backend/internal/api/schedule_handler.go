package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// ScheduleResponse is the JSON body for GET /api/v1/schedules.
type ScheduleResponse struct {
	Schedules []analyzer.ScheduleAnalysis `json:"schedules"`
}

// CronHPAResponse is the JSON body for GET /api/v1/cronhpa/{deployment}.
type CronHPAResponse struct {
	YAML   string                 `json:"yaml"`
	Config analyzer.CronHPAConfig `json:"config"`
}

// ScheduleHandler returns an HTTP handler that analyzes hourly load patterns
// for deployments.
// Query parameters:
//   - namespace: filter by Kubernetes namespace
//   - deployment: filter by Deployment name
func ScheduleHandler(lister PodLister, a *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := k8s.ListOptions{
			Namespace:  r.URL.Query().Get("namespace"),
			Deployment: r.URL.Query().Get("deployment"),
		}

		pods, err := lister.ListPods(r.Context(), opts)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to list pods", "LIST_PODS_ERROR")
			return
		}

		schedules := a.AnalyzeSchedule(pods)

		JSON(w, http.StatusOK, ScheduleResponse{
			Schedules: schedules,
		})
	}
}

// CronHPAHandler returns an HTTP handler that generates a CronHPA YAML template
// for a specific deployment.
// URL parameter:
//   - deployment: the deployment name (chi URL param)
//
// Query parameters:
//   - namespace: Kubernetes namespace (required)
func CronHPAHandler(lister PodLister, a *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deployment := chi.URLParam(r, "deployment")
		if deployment == "" {
			JSONError(w, http.StatusBadRequest, "deployment is required", "MISSING_DEPLOYMENT")
			return
		}

		namespace := r.URL.Query().Get("namespace")
		if namespace == "" {
			JSONError(w, http.StatusBadRequest, "namespace query parameter is required", "MISSING_NAMESPACE")
			return
		}

		opts := k8s.ListOptions{
			Namespace:  namespace,
			Deployment: deployment,
		}

		pods, err := lister.ListPods(r.Context(), opts)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to list pods", "LIST_PODS_ERROR")
			return
		}

		schedules := a.AnalyzeSchedule(pods)

		// Find the schedule for the requested deployment.
		var target *analyzer.ScheduleAnalysis
		for i := range schedules {
			if schedules[i].Deployment == deployment && schedules[i].Namespace == namespace {
				target = &schedules[i]
				break
			}
		}

		if target == nil {
			// No data found; generate a default config.
			target = &analyzer.ScheduleAnalysis{
				Namespace:  namespace,
				Deployment: deployment,
			}
		}

		config := analyzer.GenerateCronHPA(*target)

		yaml, err := analyzer.RenderCronHPAYAML(config)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "failed to render CronHPA YAML", "RENDER_YAML_ERROR")
			return
		}

		JSON(w, http.StatusOK, CronHPAResponse{
			YAML:   yaml,
			Config: config,
		})
	}
}
