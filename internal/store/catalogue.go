package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

type InventoryMovement struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	SKUID        string    `json:"sku_id"`
	MovementType string    `json:"movement_type"`
	Quantity     int       `json:"quantity"`
	BalanceAfter int       `json:"balance_after"`
	Note         string    `json:"note"`
	MovementDate time.Time `json:"movement_date"`
	CreatedAt    time.Time `json:"created_at"`
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
	_ = db.RecordActivity(ctx, &ActivityEvent{
		UserID:      sku.UserID,
		Kind:        "sku_edit",
		Title:       "SKU saved",
		Description: fmt.Sprintf("SKU %s was created or updated", sku.SKUID),
		EntityType:  "sku",
		EntityID:    sku.SKUID,
	})
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
	_ = db.RecordActivity(ctx, &ActivityEvent{
		UserID:      userID,
		Kind:        "sku_edit",
		Title:       "SKU deleted",
		Description: fmt.Sprintf("SKU %s was removed from the catalogue", skuID),
		EntityType:  "sku",
		EntityID:    skuID,
	})
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

func (db *DB) GetSKU(ctx context.Context, userID, skuID string) (*SKU, error) {
	var sku SKU
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, sku_id, name, unit_cost, order_cost, holding_pct, lead_time_days, selling_price, current_stock, created_at
		 FROM skus WHERE user_id = $1 AND sku_id = $2`,
		userID, skuID,
	).Scan(&sku.UserID, &sku.SKUID, &sku.Name, &sku.UnitCost, &sku.OrderCost, &sku.HoldingPct, &sku.LeadTimeDays, &sku.SellingPrice, &sku.CurrentStock, &sku.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get sku: %w", err)
	}
	return &sku, nil
}

func (db *DB) adjustSKUStockTx(ctx context.Context, tx pgx.Tx, userID, skuID string, delta int, movementType, note string, movementDate time.Time) (*SKU, error) {
	var sku SKU
	var currentStock int
	if err := tx.QueryRow(ctx, `SELECT user_id, sku_id, name, unit_cost, order_cost, holding_pct, lead_time_days, selling_price, current_stock, created_at
		 FROM skus WHERE user_id = $1 AND sku_id = $2 FOR UPDATE`, userID, skuID).Scan(&sku.UserID, &sku.SKUID, &sku.Name, &sku.UnitCost, &sku.OrderCost, &sku.HoldingPct, &sku.LeadTimeDays, &sku.SellingPrice, &currentStock, &sku.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sku not found")
		}
		return nil, fmt.Errorf("lock sku stock: %w", err)
	}

	updatedStock := currentStock + delta
	if updatedStock < 0 {
		return nil, fmt.Errorf("insufficient stock")
	}

	if _, err := tx.Exec(ctx, `UPDATE skus SET current_stock = $1 WHERE user_id = $2 AND sku_id = $3`, updatedStock, userID, skuID); err != nil {
		return nil, fmt.Errorf("update sku stock: %w", err)
	}

	if movementDate.IsZero() {
		movementDate = time.Now()
	}
	if _, err := tx.Exec(ctx, `INSERT INTO inventory_movements (user_id, sku_id, movement_type, quantity, balance_after, note, movement_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, userID, skuID, movementType, delta, updatedStock, note, movementDate); err != nil {
		return nil, fmt.Errorf("insert movement: %w", err)
	}

	sku.CurrentStock = updatedStock
	return &sku, nil
}

func (db *DB) AdjustSKUStock(ctx context.Context, userID, skuID string, delta int, movementType, note string, movementDate time.Time) (*SKU, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin stock adjustment: %w", err)
	}
	defer tx.Rollback(ctx)

	sku, err := db.adjustSKUStockTx(ctx, tx, userID, skuID, delta, movementType, note, movementDate)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit stock adjustment: %w", err)
	}

	return sku, nil
}

func (db *DB) RecordSale(ctx context.Context, userID, skuID string, quantity int, saleDate time.Time) (*SKU, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than zero")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin sale transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	sku, err := db.adjustSKUStockTx(ctx, tx, userID, skuID, -quantity, "sale", fmt.Sprintf("Sold on %s", saleDate.Format("2006-01-02")), saleDate)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO sales_entries (user_id, sku_id, date, quantity) VALUES ($1, $2, $3, $4)`,
		userID, skuID, saleDate, quantity,
	); err != nil {
		return nil, fmt.Errorf("insert sales entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit sale transaction: %w", err)
	}
	_ = db.RecordActivity(ctx, &ActivityEvent{
		UserID:      userID,
		Kind:        "sku_edit",
		Title:       "Sale logged",
		Description: fmt.Sprintf("%s sold %d units", skuID, quantity),
		EntityType:  "sku",
		EntityID:    skuID,
	})

	return sku, nil
}

func (db *DB) RecordReplenishment(ctx context.Context, userID, skuID string, quantity int, movementDate time.Time, note string) (*SKU, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than zero")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin replenishment transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	sku, err := db.adjustSKUStockTx(ctx, tx, userID, skuID, quantity, "replenish", note, movementDate)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replenishment transaction: %w", err)
	}
	_ = db.RecordActivity(ctx, &ActivityEvent{
		UserID:      userID,
		Kind:        "sku_edit",
		Title:       "Stock replenished",
		Description: fmt.Sprintf("%s replenished by %d units", skuID, quantity),
		EntityType:  "sku",
		EntityID:    skuID,
	})

	return sku, nil
}

func (db *DB) GetInventoryMovements(ctx context.Context, userID, skuID string) ([]InventoryMovement, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, sku_id, movement_type, quantity, balance_after, note, movement_date, created_at
		 FROM inventory_movements WHERE user_id = $1 AND sku_id = $2 ORDER BY movement_date DESC, created_at DESC`,
		userID, skuID,
	)
	if err != nil {
		return nil, fmt.Errorf("get inventory movements: %w", err)
	}
	defer rows.Close()

	var movements []InventoryMovement
	for rows.Next() {
		var movement InventoryMovement
		if err := rows.Scan(&movement.ID, &movement.UserID, &movement.SKUID, &movement.MovementType, &movement.Quantity, &movement.BalanceAfter, &movement.Note, &movement.MovementDate, &movement.CreatedAt); err != nil {
			return nil, err
		}
		movements = append(movements, movement)
	}
	return movements, nil
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
