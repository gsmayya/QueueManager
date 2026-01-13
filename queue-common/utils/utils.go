package utils

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is a consistent JSON error envelope returned by handlers.
type ErrorResponse struct {
	Error string `json:"error"`
}

// RespondWithJSON writes a JSON response with the given status code.
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// RespondWithError writes a JSON error response with the given status code.
func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	RespondWithJSON(w, statusCode, ErrorResponse{Error: message})
}
