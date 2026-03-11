// Package store provides PostgreSQL storage for users and analysis reports.
//
// It manages the connection pool, runs schema migrations on startup,
// and exposes repository methods that the API layer calls. The store
// never touches HTTP or business logic — it only persists and retrieves data.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool.Pool and provides repository methods.
type DB struct {
	Pool *pgxpool.Pool
}

// New opens a connection pool and runs migrations.
// dsn example: "postgres://user:pass@localhost:5432/inventory?sslmode=disable"
func New(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	db := &DB{Pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Close releases the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// migrate runs the schema DDL idempotently.
func (db *DB) migrate(ctx context.Context) error {
	ddl := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS users (
		id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email      TEXT UNIQUE NOT NULL,
		password   TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE TABLE IF NOT EXISTS reports (
		id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title         TEXT NOT NULL DEFAULT '',
		service_level DOUBLE PRECISION NOT NULL DEFAULT 0.95,
		sim_runs      INTEGER NOT NULL DEFAULT 500,
		sim_weeks     INTEGER NOT NULL DEFAULT 52,
		sku_count     INTEGER NOT NULL DEFAULT 0,
		warnings      JSONB NOT NULL DEFAULT '[]',
		results       JSONB NOT NULL DEFAULT '[]',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_reports_user_id ON reports(user_id);
	CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at DESC);
	`
	_, err := db.Pool.Exec(ctx, ddl)
	return err
}
