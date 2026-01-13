package db

import (
	"context"
	"database/sql"
)

// EnsureMasterSchema creates the master_db schema if it doesn't exist.
// Safe to run on every startup.
func EnsureMasterSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto;`,
		`CREATE TABLE IF NOT EXISTS entities (
			id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			name         text NOT NULL,
			phone        text NOT NULL,
			email        text,
			joining_date timestamptz NOT NULL DEFAULT now(),
			CONSTRAINT entities_name_phone_uniq UNIQUE (name, phone)
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id       text NOT NULL UNIQUE,
			name          text NOT NULL,
			email         text NOT NULL UNIQUE,
			password_hash text NOT NULL,
			created_at    timestamptz NOT NULL DEFAULT now()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_entities_joining_date ON entities(joining_date);`,
		`CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
