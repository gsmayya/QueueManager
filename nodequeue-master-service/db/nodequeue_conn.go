package db

import (
	"database/sql"
	"os"
	"time"
)

// OpenNodequeueFromEnv opens a connection to the nodequeue database for room/resource admin APIs.
//
// Env vars:
// - NODEQUEUE_DB_HOST / PORT / NAME / USER / PASSWORD / SSLMODE
//
// Defaults:
// - NODEQUEUE_DB_NAME defaults to "nodequeue"
// - NODEQUEUE_DB_* fall back to MASTER_DB_* when not explicitly set (useful in docker-compose)
func OpenNodequeueFromEnv() (*sql.DB, error) {
	// Temporarily map NODEQUEUE_DB_* into the expected MASTER_DB_* slots by constructing a Config.
	cfg := Config{
		Host:     firstNonEmpty(os.Getenv("NODEQUEUE_DB_HOST"), os.Getenv("MASTER_DB_HOST")),
		Port:     firstNonEmpty(os.Getenv("NODEQUEUE_DB_PORT"), os.Getenv("MASTER_DB_PORT")),
		Name:     firstNonEmpty(os.Getenv("NODEQUEUE_DB_NAME"), "nodequeue"),
		User:     firstNonEmpty(os.Getenv("NODEQUEUE_DB_USER"), os.Getenv("MASTER_DB_USER")),
		Password: firstNonEmpty(os.Getenv("NODEQUEUE_DB_PASSWORD"), os.Getenv("MASTER_DB_PASSWORD")),
		SSLMode:  firstNonEmpty(os.Getenv("NODEQUEUE_DB_SSLMODE"), os.Getenv("MASTER_DB_SSLMODE"), "disable"),
	}
	if !cfg.Enabled() {
		return nil, nil
	}

	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, err
	}
	// Reuse same tuning + ping logic as OpenFromEnv.
	// (kept inline to avoid exposing internal helpers)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
