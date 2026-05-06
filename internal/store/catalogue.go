package store

import (
	"context"
	"fmt"
	"time"
)

type SKU struct {
	UserID       string    `json:"user_id"`
	SKUID        string    `json:"sku_id"`
	Name         string    `json:"name"`
	UnitCost     float64   `json:"unit_cost"`
	OrderCost    float64   `json:"order_cost"`
	HoldingPct   float64   `json:"holding_pct"`
	LeadTimeDays int       `json:"lead_time_days"`
	SellingPrice float64   `json:"selling_price"`
	CurrentStock int       `json:"current_stock"`
	CreatedAt    time.Time `json:"created_at"`
}

type SalesEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	SKUID     string    `json:"sku_id"`
	Date      time.Time `json:"date"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) CreateSKU(ctx context.Context, sku *SKU) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO skus (user_id, sku_id, name, unit_cost, order_cost, holding_pct, lead_time_days, selling_price, current_stock)
 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
 ON CONFLICT (user_id, sku_id) DO UPDATE SET
 name = EXCLUDED.name, unit_cost = EXCLUDED.unit_cost, order_cost = EXCLUDED.order_cost,
 holding_pct = EXCLUDED.holding_pct, lead_time_days = EXCLUDED.lead_time_days, selling_price = EXCLUDED.selling_price, current_stock = EXCLUDED.current_stock`,
		sku.UserID, sku.SKUID, sku.Name, sku.UnitCost, sku.OrderCost, sku.HoldingPct, sku.LeadTimeDays, sku.SellingPrice, sku.CurrentStock,
	)
	if err != nil {
		return fmt.Errorf("create/update sku: %w", err)
	}
	return nil
}

func (db *DB) GetSKUs(ctx context.Context, userID string) ([]SKU, error) {
	rows, err := db.Pool.Query(ctx, `SELECT user_id, sku_id, name, unit_cost, order_cost, holding_pct, lead_time_days, selling_price, current_stock, created_at FROM skus WHERE user_id = $1 ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get skus: %w", err)
	}
	defer rows.Close()

	var skus []SKU
	for rows.Next() {
		var s SKU
		if err := rows.Scan(&s.UserID, &s.SKUID, &s.Name, &s.UnitCost, &s.OrderCost, &s.HoldingPct, &s.LeadTimeDays, &s.SellingPrice, &s.CurrentStock, &s.CreatedAt); err != nil {
			return nil, err
		}
		skus = append(skus, s)
	}
	return skus, nil
}

func (db *DB) DeleteSKU(ctx context.Context, userID, skuID string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM skus WHERE user_id = $1 AND sku_id = $2`, userID, skuID)
	if err != nil {
		return fmt.Errorf("delete sku: %w", err)
	}
	return nil
}

func (db *DB) AddSalesEntry(ctx context.Context, entry *SalesEntry) error {
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO sales_entries (user_id, sku_id, date, quantity) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		entry.UserID, entry.SKUID, entry.Date, entry.Quantity,
	).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("add sales entry: %w", err)
	}
	return nil
}

func (db *DB) GetSalesEntries(ctx context.Context, userID string) ([]SalesEntry, error) {
	rows, err := db.Pool.Query(ctx, `SELECT id, user_id, sku_id, date, quantity, created_at FROM sales_entries WHERE user_id = $1 ORDER BY date DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get sales entries: %w", err)
	}
	defer rows.Close()

	var entries []SalesEntry
	for rows.Next() {
		var e SalesEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.SKUID, &e.Date, &e.Quantity, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
