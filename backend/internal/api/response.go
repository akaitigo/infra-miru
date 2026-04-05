package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a JSON error response body.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// JSON writes data as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, `{"error":"internal server error","code":"ENCODE_ERROR"}`, http.StatusInternalServerError)
	}
}

// JSONError writes an error response with the given status code, message, and error code.
func JSONError(w http.ResponseWriter, status int, message, code string) {
	JSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}
