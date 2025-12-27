package main

import (
	"encoding/json"
	"net/http"
)

// Entity is the domain object referenced by a Node.
// In this service it's intentionally minimal (just a name) and is embedded in API payloads.
type Entity struct {
	Name string `json:"name"`
}

// corsMiddleware wraps a handler with permissive CORS headers for browser-based clients.
//
// It also short-circuits OPTIONS preflight requests with HTTP 200.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// respondWithJSON writes a JSON response with the given status code.
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError writes a JSON error response with the given status code.
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, ErrorResponse{Error: message})
}
