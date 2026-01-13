package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"nodequeue-master-service/db"
	"nodequeue-master-service/handlers"
	"nodequeue-master-service/store"
)

func main() {
	dbConn, err := db.OpenFromEnv()
	if err != nil {
		log.Fatalf("[DB] failed to connect: %v", err)
	}
	if dbConn == nil {
		log.Fatal("[DB] missing MASTER_DB_* env vars (MASTER_DB_HOST/PORT/USER required)")
	}
	defer dbConn.Close()

	nodequeueConn, err := db.OpenNodequeueFromEnv()
	if err != nil {
		log.Fatalf("[DB] failed to connect to nodequeue db: %v", err)
	}
	if nodequeueConn == nil {
		log.Fatal("[DB] missing NODEQUEUE_DB_* env vars (or MASTER_DB_* fallback) for rooms APIs")
	}
	defer nodequeueConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.EnsureMasterSchema(ctx, dbConn); err != nil {
		log.Fatalf("[DB] failed to ensure master_db schema: %v", err)
	}

	st := store.NewPostgresStore(dbConn)
	roomStore := store.NewNodequeuePostgresStore(nodequeueConn)
	svc := handlers.NewService(st, roomStore)

	setupRoutes(svc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	addr := fmt.Sprintf(":%s", port)

	log.Printf("Starting master-service on %s", addr)
	log.Println("API Endpoints:")
	log.Println("  GET    /health")
	log.Println("  POST   /entities")
	log.Println("  GET    /entities")
	log.Println("  GET    /entities/{id}")
	log.Println("  PUT    /entities/{id}")
	log.Println("  DELETE /entities/{id}")
	log.Println("  POST   /users")
	log.Println("  GET    /users")
	log.Println("  GET    /users/{id}")
	log.Println("  PUT    /users/{id}")
	log.Println("  DELETE /users/{id}")
	log.Println("  POST   /rooms")
	log.Println("  GET    /rooms")
	log.Println("  GET    /rooms/{id}")
	log.Println("  PUT    /rooms/{id}")
	log.Println("  DELETE /rooms/{id} (soft delete)")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
