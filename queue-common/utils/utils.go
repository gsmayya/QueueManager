package utils

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return false
	}
	return true
}

func ParseUUIDParam(w http.ResponseWriter, id string) bool {
	if _, err := uuid.Parse(id); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return false
	}
	return true
}
