package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	store "queue-common/store"
	"queue-service/queueservice"

	"queue-common/db"
)

// main is the program entry point. It initializes resources, registers routes,
// and starts the HTTP server.
func main() {
	// Optional DB connection (best-effort). If env vars are not set or DB is down, we run in-memory.
	dbConn, err := db.OpenFromEnv(getDBConfigFromEnv())
	if err != nil {
		log.Printf("[DB] disabled (failed to connect): %v", err)
	}
	if dbConn != nil {
		defer dbConn.Close()
	}

	var nodestore store.NodeStore
	if dbConn != nil {
		nodestore = store.NewNodeStore(dbConn)
	}
	var resourcestore store.ResStore
	if dbConn != nil {
		resourcestore = store.NewResStore(dbConn)
	}
	var schedstore store.ScheduleStore
	if dbConn != nil {
		schedstore = store.NewScheduleStore(dbConn)
	}

	// Initialize queue service
	queueService := queueservice.NewQueueServiceWithStores(nodestore, resourcestore, schedstore)
	// Load resources from config (or fall back to defaults).
	resources := setupResources(queueService, nodestore, resourcestore)
	log.Printf("Initialized %d resources", len(resources))

	// Restore nodes + queue membership from DB (best-effort).
	if nodestore != nil {
		if err := queueService.RestoreFromStore(context.Background()); err != nil {
			log.Printf("[DB] restore state failed (continuing with empty node state): %v", err)
		}
	}

	// Start in-process scheduler loop (no-op if scheduling stores are not configured).
	tickSeconds := 2
	if v := os.Getenv("SCHEDULER_TICK_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			tickSeconds = n
		}
	}
	ticker := time.NewTicker(time.Duration(tickSeconds) * time.Second)
	go func() {
		for t := range ticker.C {
			queueService.RunDueSchedules(context.Background(), t)
			queueService.ExpiryScan(context.Background(), t)
			queueService.DelayFlagScan(context.Background(), t)
		}
	}()

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

func getDBConfigFromEnv() db.Config {
	sslmode := os.Getenv("MAIN_DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	name := os.Getenv("MAIN_DB_NAME")
	if name == "" {
		name = "nodequeue"
	}
	return db.Config{
		Host:     os.Getenv("MAIN_DB_HOST"),
		Port:     os.Getenv("MAIN_DB_PORT"),
		Name:     name,
		User:     os.Getenv("MAIN_DB_USER"),
		Password: os.Getenv("MAIN_DB_PASSWORD"),
		SSLMode:  sslmode,
	}
}
