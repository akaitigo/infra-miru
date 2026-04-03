package api

import (
	"net/http"

	"github.com/akaitigo/infra-miru/backend/internal/analyzer"
	"github.com/akaitigo/infra-miru/backend/internal/cost"
	"github.com/akaitigo/infra-miru/backend/internal/k8s"
)

// RecommendationResponse is the JSON body for GET /api/v1/recommendations.
type RecommendationResponse struct {
	Recommendations []cost.Recommendation `json:"recommendations"`
}

// RecommendationHandler returns an HTTP handler that lists cost optimization
// recommendations for over-provisioned Deployments.
// Query parameters:
//   - namespace: filter by Kubernetes namespace
//   - deployment: filter by Deployment name
func RecommendationHandler(lister PodLister, a *analyzer.Analyzer, c *cost.Calculator) http.HandlerFunc {
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
		recommendations := c.CalculateRecommendations(analyses)

		JSON(w, http.StatusOK, RecommendationResponse{
			Recommendations: recommendations,
		})
	}
}
