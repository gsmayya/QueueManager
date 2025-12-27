package main

import (
	"log"
	"net/http"
	"strings"

	"nodequeue-service/queueservice"
	"nodequeue-service/resource"
)

// setupRoutes registers the HTTP routes for the NodeQueue service.
//
// Note: net/http's DefaultServeMux is used for simplicity.
func setupRoutes(qs *queueservice.QueueService) {
	http.HandleFunc("/nodes", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			qs.CreateNodeHandler(w, r)
		case http.MethodGet:
			qs.ListNodesHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/nodes/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/nodes/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			qs.ListNodesHandler(w, r)
			return
		}

		nodeID := parts[0]

		// Handle sub-routes: /nodes/{id}/move or /nodes/{id}/complete
		if len(parts) == 2 {
			switch parts[1] {
			case "move":
				if r.Method == http.MethodPost {
					qs.MoveNodeHandler(w, r, nodeID)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			case "allocate":
				if r.Method == http.MethodPost {
					qs.AllocateNodeHandler(w, r, nodeID)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			case "complete":
				if r.Method == http.MethodPost {
					qs.CompleteNodeHandler(w, r, nodeID)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
		}

		// Handle GET /nodes/{id}
		if r.Method == http.MethodGet {
			qs.GetNodeHandler(w, r, nodeID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/resources", corsMiddleware(qs.ListResourcesHandler))
}

func setupResources(fileName string, queueService *queueservice.QueueService) []*resource.Resource {
	resources := resource.LoadResources(fileName)
	for _, r := range resources {
		queueService.AddResource(r)
		log.Printf("Initialized resource %s with capacity %d", r.ID, r.Capacity)
	}
	return resources
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
