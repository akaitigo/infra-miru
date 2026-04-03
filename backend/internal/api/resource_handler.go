package api

import (
	"context"
	"net/http"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// PodLister abstracts Kubernetes Pod listing so that handlers can be tested
// without a real cluster.
type PodLister interface {
	ListPods(ctx context.Context, opts k8s.ListOptions) ([]k8s.PodResource, error)
}

// ResourceResponse is the JSON body for GET /api/v1/resources.
type ResourceResponse struct {
	Pods        []analyzer.ResourceAnalysis  `json:"pods"`
	Deployments []analyzer.DeploymentSummary `json:"deployments"`
}

// ResourceHandler returns an HTTP handler that lists Pod resource utilization.
// Query parameters:
//   - namespace: filter by Kubernetes namespace
//   - deployment: filter by Deployment name
func ResourceHandler(lister PodLister, a *analyzer.Analyzer) http.HandlerFunc {
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

		analyses := a.AnalyzeResources(pods)
		deployments := a.AggregateByDeployment(analyses)

		JSON(w, http.StatusOK, ResourceResponse{
			Pods:        analyses,
			Deployments: deployments,
		})
	}
}
