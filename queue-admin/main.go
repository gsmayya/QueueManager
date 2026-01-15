package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"queue-admin/handlers"
	"queue-common/db"
	"queue-common/store"
)

func main() {
	dbConn, err := db.OpenFromEnv(getDBConfigFromEnv())
	if err != nil {
		log.Fatalf("[DB] failed to connect: %v", err)
	}
	if dbConn == nil {
		log.Fatal("[DB] missing MASTER_DB_* env vars (MASTER_DB_HOST/PORT/USER required)")
	}
	defer dbConn.Close()

	nodequeueConn, err := db.OpenFromEnv(getNodeQueueConfigFromEnv())
	if err != nil {
		log.Fatalf("[DB] failed to connect to nodequeue db: %v", err)
	}
	if nodequeueConn == nil {
		log.Fatal("[DB] missing NODEQUEUE_DB_* env vars (or MASTER_DB_* fallback) for rooms APIs")
	}
	defer nodequeueConn.Close()

	userStore := store.NewUserStore(dbConn)
	entityStore := store.NewEntityStore(dbConn)
	roomStore := store.NewResourceStore(nodequeueConn)

	svc := handlers.NewService(entityStore, userStore, roomStore)

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

func getDBConfigFromEnv() db.Config {
	sslmode := os.Getenv("MAIN_DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	name := os.Getenv("MAIN_DB_NAME")
	if name == "" {
		name = "master_db"
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

func getNodeQueueConfigFromEnv() db.Config {
	sslmode := os.Getenv("NODEQUEUE_DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	name := os.Getenv("NODEQUEUE_DB_NAME")
	if name == "" {
		name = "nodequeue"
	}
	return db.Config{
		Host:     os.Getenv("NODEQUEUE_DB_HOST"),
		Port:     os.Getenv("NODEQUEUE_DB_PORT"),
		Name:     name,
		User:     os.Getenv("NODEQUEUE_DB_USER"),
		Password: os.Getenv("NODEQUEUE_DB_PASSWORD"),
		SSLMode:  sslmode,
	}
}
