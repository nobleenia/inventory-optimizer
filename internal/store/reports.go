package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// Report represents a persisted analysis report.
type Report struct {
	ID           string             `json:"id"`
	UserID       string             `json:"user_id"`
	Title        string             `json:"title"`
	ServiceLevel float64            `json:"service_level"`
	SimRuns      int                `json:"sim_runs"`
	SimWeeks     int                `json:"sim_weeks"`
	SKUCount     int                `json:"sku_count"`
	Warnings     []string           `json:"warnings"`
	Results      []models.SKUReport `json:"results"`
	CreatedAt    time.Time          `json:"created_at"`
}

// ErrReportNotFound is returned when a report lookup finds no match.
var ErrReportNotFound = errors.New("report not found")

// CreateReport persists a new analysis report for a user.
func (db *DB) CreateReport(ctx context.Context, r *Report) error {
	warningsJSON, err := json.Marshal(r.Warnings)
	if err != nil {
		return fmt.Errorf("marshal warnings: %w", err)
	}

	resultsJSON, err := json.Marshal(r.Results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	err = db.Pool.QueryRow(ctx,
		`INSERT INTO reports (user_id, title, service_level, sim_runs, sim_weeks, sku_count, warnings, results)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		r.UserID, r.Title, r.ServiceLevel, r.SimRuns, r.SimWeeks, r.SKUCount, warningsJSON, resultsJSON,
	).Scan(&r.ID, &r.CreatedAt)

	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	return nil
}

// GetReport retrieves a report by ID, scoped to a user.
func (db *DB) GetReport(ctx context.Context, userID, reportID string) (*Report, error) {
	r := &Report{}
	var warningsJSON, resultsJSON []byte

	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, title, service_level, sim_runs, sim_weeks, sku_count, warnings, results, created_at
		 FROM reports WHERE id = $1 AND user_id = $2`,
		reportID, userID,
	).Scan(
		&r.ID, &r.UserID, &r.Title, &r.ServiceLevel, &r.SimRuns, &r.SimWeeks,
		&r.SKUCount, &warningsJSON, &resultsJSON, &r.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get report: %w", err)
	}

	if err := json.Unmarshal(warningsJSON, &r.Warnings); err != nil {
		return nil, fmt.Errorf("unmarshal warnings: %w", err)
	}
	if err := json.Unmarshal(resultsJSON, &r.Results); err != nil {
		return nil, fmt.Errorf("unmarshal results: %w", err)
	}

	return r, nil
}

// ListReports returns all reports for a user, newest first.
func (db *DB) ListReports(ctx context.Context, userID string, limit, offset int) ([]Report, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get total count.
	var total int
	err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM reports WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, title, service_level, sim_runs, sim_weeks, sku_count, warnings, created_at
		 FROM reports WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		r := Report{}
		var warningsJSON []byte
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.Title, &r.ServiceLevel, &r.SimRuns, &r.SimWeeks,
			&r.SKUCount, &warningsJSON, &r.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan report: %w", err)
		}
		if err := json.Unmarshal(warningsJSON, &r.Warnings); err != nil {
			r.Warnings = []string{}
		}
		reports = append(reports, r)
	}

	return reports, total, nil
}

// DeleteReport removes a report, scoped to a user.
func (db *DB) DeleteReport(ctx context.Context, userID, reportID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM reports WHERE id = $1 AND user_id = $2`,
		reportID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete report: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReportNotFound
	}
	return nil
}
