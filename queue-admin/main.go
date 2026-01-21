package main

import (
	"fmt"
	"net/http"
	"os"

	"queue-admin/handlers"
	"queue-common/db"
	"queue-common/logging"
	"queue-common/store"
)

func main() {
	dbConn, err := db.OpenFromEnv(getDBConfigFromEnv())
	if err != nil {
		logging.Errorf("[DB] failed to connect: %v", err)
		os.Exit(1)
	}
	if dbConn == nil {
		logging.Errorf("[DB] missing MASTER_DB_* env vars (MASTER_DB_HOST/PORT/USER required)")
		os.Exit(1)
	}
	defer dbConn.Close()

	nodequeueConn, err := db.OpenFromEnv(getNodeQueueConfigFromEnv())
	if err != nil {
		logging.Errorf("[DB] failed to connect to nodequeue db: %v", err)
		os.Exit(1)
	}
	if nodequeueConn == nil {
		logging.Errorf("[DB] missing NODEQUEUE_DB_* env vars (or MASTER_DB_* fallback) for rooms APIs")
		os.Exit(1)
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

	logging.Infof("Starting master-service on %s", addr)
	logging.Infof("API Endpoints:")
	logging.Infof("  GET    /health")
	logging.Infof("  POST   /entities")
	logging.Infof("  GET    /entities")
	logging.Infof("  GET    /entities/{id}")
	logging.Infof("  PUT    /entities/{id}")
	logging.Infof("  DELETE /entities/{id}")
	logging.Infof("  POST   /users")
	logging.Infof("  GET    /users")
	logging.Infof("  GET    /users/{id}")
	logging.Infof("  PUT    /users/{id}")
	logging.Infof("  DELETE /users/{id}")
	logging.Infof("  POST   /rooms")
	logging.Infof("  GET    /rooms")
	logging.Infof("  GET    /rooms/{id}")
	logging.Infof("  PUT    /rooms/{id}")
	logging.Infof("  DELETE /rooms/{id} (soft delete)")

	if err := http.ListenAndServe(addr, nil); err != nil {
		logging.Errorf("Server failed to start: %v", err)
		os.Exit(1)
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
