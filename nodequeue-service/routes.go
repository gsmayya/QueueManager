package main

import (
	"log"
	"net/http"
	"strings"
)

// setupRoutes registers the HTTP routes for the NodeQueue service.
//
// Note: net/http's DefaultServeMux is used for simplicity.
func setupRoutes(qs *QueueService) {
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

func setupResources(fileName string, queueService *QueueService) []resourceConfig {
	resources := loadResources(fileName)
	// Add resources to the service
	for _, r := range resources {
		resource := NewResource(r.id, r.capacity)
		queueService.AddResource(resource)
		log.Printf("Initialized resource %s with capacity %d", r.id, r.capacity)
	}
	return resources
}
