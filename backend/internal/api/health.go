// Package api provides HTTP handlers, routing, and server infrastructure
// for the Infra-Miru API.
package api

import (
	"net/http"
)

// HealthResponse represents the response body of the health check endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler returns an HTTP handler that responds with the server health status.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		JSON(w, http.StatusOK, HealthResponse{Status: "ok"})
	}
}
