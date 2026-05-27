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
		preferred_currency TEXT NOT NULL DEFAULT 'USD',
		country_code       TEXT NOT NULL DEFAULT '',
		business_type      TEXT NOT NULL DEFAULT 'retail',
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
		tags          JSONB NOT NULL DEFAULT '[]',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_reports_user_id ON reports(user_id);
	CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at DESC);

	CREATE TABLE IF NOT EXISTS subscriptions (
		user_id                UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		stripe_customer_id     TEXT NOT NULL DEFAULT '',
		stripe_subscription_id TEXT NOT NULL DEFAULT '',
		status                 TEXT NOT NULL DEFAULT 'inactive', /* active, inactive, past_due, canceled */
		current_period_end     TIMESTAMPTZ,
		updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE TABLE IF NOT EXISTS skus (
		user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		sku_id         TEXT NOT NULL,
		name           TEXT NOT NULL DEFAULT '',
		unit_cost      DOUBLE PRECISION NOT NULL DEFAULT 0,
		order_cost     DOUBLE PRECISION NOT NULL DEFAULT 0,
		holding_pct    DOUBLE PRECISION NOT NULL DEFAULT 0.25,
		lead_time_days INTEGER NOT NULL DEFAULT 14,
                selling_price DOUBLE PRECISION NOT NULL DEFAULT 0,
                current_stock INTEGER NOT NULL DEFAULT 0,
		created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (user_id, sku_id)
	);

	CREATE TABLE IF NOT EXISTS sales_entries (
		id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		sku_id     TEXT NOT NULL,
		date       DATE NOT NULL,
		quantity   INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		FOREIGN KEY (user_id, sku_id) REFERENCES skus(user_id, sku_id) ON DELETE CASCADE
	);

	
        CREATE TABLE IF NOT EXISTS generated_records (
                id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                template_name TEXT NOT NULL,
                file_path     TEXT NOT NULL,
                records_count INTEGER NOT NULL DEFAULT 0,
                created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
        );
        CREATE INDEX IF NOT EXISTS idx_generated_records_user_id ON generated_records(user_id);
        
        CREATE INDEX IF NOT EXISTS idx_sales_entries_user_sku ON sales_entries(user_id, sku_id);
	CREATE INDEX IF NOT EXISTS idx_sales_entries_date ON sales_entries(date);

        CREATE TABLE IF NOT EXISTS generated_records (
                id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                template_name TEXT NOT NULL,
                file_path     TEXT NOT NULL,
                records_count INTEGER NOT NULL DEFAULT 0,
                created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
        );
        CREATE INDEX IF NOT EXISTS idx_generated_records_user_id ON generated_records(user_id);

	`
	db.Pool.Exec(ctx, `ALTER TABLE skus ADD COLUMN IF NOT EXISTS selling_price DOUBLE PRECISION NOT NULL DEFAULT 0;`)
	db.Pool.Exec(ctx, `ALTER TABLE skus ADD COLUMN IF NOT EXISTS current_stock INTEGER NOT NULL DEFAULT 0;`)
	db.Pool.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_currency TEXT NOT NULL DEFAULT 'USD';`)
	db.Pool.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS country_code TEXT NOT NULL DEFAULT '';`)
	db.Pool.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS business_type TEXT NOT NULL DEFAULT 'retail';`)
	db.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS inventory_movements (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		sku_id TEXT NOT NULL,
		movement_type TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		balance_after INTEGER NOT NULL,
		note TEXT NOT NULL DEFAULT '',
		movement_date TIMESTAMPTZ NOT NULL DEFAULT now(),
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		FOREIGN KEY (user_id, sku_id) REFERENCES skus(user_id, sku_id) ON DELETE CASCADE
	);`)
	db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_inventory_movements_user_sku ON inventory_movements(user_id, sku_id);`)
	db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_inventory_movements_date ON inventory_movements(movement_date DESC);`)

	// Ensure reports.tags exists for newer code paths
	db.Pool.Exec(ctx, `ALTER TABLE reports ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]';`)

	// Saved filters table for persisting user filter sets
	db.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS saved_filters (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		params JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);`)
	db.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS notification_settings (
		user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		enabled BOOLEAN NOT NULL DEFAULT false,
		frequency TEXT NOT NULL DEFAULT 'daily',
		scheduled_time TEXT NOT NULL DEFAULT '09:00',
		email_override TEXT NOT NULL DEFAULT '',
		timezone TEXT NOT NULL DEFAULT 'UTC',
		last_sent_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);`)
	db.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS notifications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind TEXT NOT NULL DEFAULT 'replenishment',
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		report_id TEXT NOT NULL DEFAULT '',
		read_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);`)
	db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_notifications_user_created_at ON notifications(user_id, created_at DESC);`)
	db.Pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS activity_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		entity_type TEXT NOT NULL DEFAULT '',
		entity_id TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);`)
	db.Pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_activity_events_user_created_at ON activity_events(user_id, created_at DESC);`)
	_, err := db.Pool.Exec(ctx, ddl)
	return err
}
