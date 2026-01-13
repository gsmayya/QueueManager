package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

func ConfigFromEnv() Config {
	sslmode := os.Getenv("MASTER_DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	name := os.Getenv("MASTER_DB_NAME")
	if name == "" {
		name = "master_db"
	}
	return Config{
		Host:     os.Getenv("MASTER_DB_HOST"),
		Port:     os.Getenv("MASTER_DB_PORT"),
		Name:     name,
		User:     os.Getenv("MASTER_DB_USER"),
		Password: os.Getenv("MASTER_DB_PASSWORD"),
		SSLMode:  sslmode,
	}
}

func (c Config) Enabled() bool {
	return c.Host != "" && c.Port != "" && c.Name != "" && c.User != ""
}

func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

func isDatabaseMissing(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "3D000"
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 3D000") || (strings.Contains(msg, "database") && strings.Contains(msg, "does not exist"))
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func ensureDatabaseExists(ctx context.Context, cfg Config) error {
	adminCfg := cfg
	adminCfg.Name = "postgres"
	adminDB, err := sql.Open("pgx", adminCfg.DSN())
	if err != nil {
		return err
	}
	defer adminDB.Close()
	adminDB.SetMaxOpenConns(2)
	adminDB.SetMaxIdleConns(1)
	adminDB.SetConnMaxLifetime(5 * time.Minute)
	if err := adminDB.PingContext(ctx); err != nil {
		return err
	}

	var exists bool
	if err := adminDB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, cfg.Name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = adminDB.ExecContext(ctx, `CREATE DATABASE `+quoteIdent(cfg.Name))
	return err
}

func OpenFromEnv() (*sql.DB, error) {
	cfg := ConfigFromEnv()
	if !cfg.Enabled() {
		return nil, nil
	}

	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		if isDatabaseMissing(err) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err2 := ensureDatabaseExists(ctx, cfg); err2 == nil {
				db2, err3 := sql.Open("pgx", cfg.DSN())
				if err3 != nil {
					return nil, err3
				}
				db2.SetMaxOpenConns(10)
				db2.SetMaxIdleConns(5)
				db2.SetConnMaxLifetime(30 * time.Minute)
				if err3 := db2.Ping(); err3 == nil {
					return db2, nil
				}
				_ = db2.Close()
				return nil, err3
			}
		}
		return nil, err
	}
	return db, nil
}
