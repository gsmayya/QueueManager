package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"nodequeue-service/queueservice"
)

// main is the program entry point. It initializes resources, registers routes,
// and starts the HTTP server.
func main() {
	// Initialize queue service
	queueService := queueservice.NewQueueService()

	// Load resources from config (or fall back to defaults).
	resources := setupResources("config.txt", queueService)
	log.Printf("Initialized %d resources", len(resources))

	// Setup HTTP routes
	setupRoutes(queueService)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)
	log.Println("API Endpoints:")
	log.Println("  POST   /nodes - Create a new node")
	log.Println("  GET    /nodes - List all nodes")
	log.Println("  GET    /nodes/{id} - Get a specific node")
	log.Println("  POST   /nodes/{id}/move - Move a node to another resource")
	log.Println("  POST   /nodes/{id}/allocate - Allocate a waiting node into the service queue (capacity enforced)")
	log.Println("  POST   /nodes/{id}/complete - Complete a node")
	log.Println("  GET    /resources - List all resources")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
