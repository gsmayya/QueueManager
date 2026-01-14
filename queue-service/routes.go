package main

import (
	"context"
	"log"
	"net/http"
	"queue-common/models"
	"strings"

	"queue-service/queueservice"

	"queue-service/store"
)

// setupRoutes registers the HTTP routes for the NodeQueue service.
//
// Note: net/http's DefaultServeMux is used for simplicity.
func setupRoutes(qs *queueservice.QueueService) {
	http.HandleFunc("/nodes/metrics", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		qs.NodesMetricsHandler(w, r)
	}))

	http.HandleFunc("/resources/metrics", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		qs.ResourcesMetricsHandler(w, r)
	}))

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
		path, ok := strings.CutPrefix(r.URL.Path, "/nodes/")
		if !ok || path == "" {
			qs.ListNodesHandler(w, r)
			return
		}

		nodeID, rest, _ := strings.Cut(path, "/")
		if nodeID == "" {
			qs.ListNodesHandler(w, r)
			return
		}

		// Handle sub-routes: /nodes/{id}/move or /nodes/{id}/complete
		if rest != "" && !strings.Contains(rest, "/") {
			switch rest {
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

func setupResources(fileName string, queueService *queueservice.QueueService, store store.Store) []*models.Resource {
	// Prefer DB resources when available, but fall back to local defaults if DB isn't configured/reachable.
	if store != nil {
		if dbResources, err := store.ListResources(context.Background()); err == nil && len(dbResources) > 0 {
			for _, r := range dbResources {
				queueService.AddResource(r)
				log.Printf("Initialized resource %s with capacity %d (from DB)", r.ID, r.Capacity)
			}
			return dbResources
		} else if err != nil {
			log.Printf("[DB] load resources failed, falling back to defaults: %v", err)
		}
	}

	resources := models.LoadResources(fileName)
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
