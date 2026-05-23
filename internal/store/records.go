package store

import (
	"context"
	"fmt"
	"time"
)

type GeneratedRecord struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	TemplateName string    `json:"template_name"`
	FilePath     string    `json:"-"`
	RecordsCount int       `json:"records_count"`
	CreatedAt    time.Time `json:"created_at"`
}

func (db *DB) SaveGeneratedRecord(ctx context.Context, record *GeneratedRecord) error {
	err := db.Pool.QueryRow(ctx,
		"INSERT INTO generated_records (user_id, template_name, file_path, records_count) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		record.UserID, record.TemplateName, record.FilePath, record.RecordsCount,
	).Scan(&record.ID, &record.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert generated record: %w", err)
	}
	return nil
}

func (db *DB) GetGeneratedRecords(ctx context.Context, userID string) ([]GeneratedRecord, error) {
	rows, err := db.Pool.Query(ctx, 
		"SELECT id, user_id, template_name, file_path, records_count, created_at FROM generated_records WHERE user_id = $1 ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, fmt.Errorf("get generated records: %w", err)
	}
	defer rows.Close()

	var records []GeneratedRecord
	for rows.Next() {
		var r GeneratedRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.TemplateName, &r.FilePath, &r.RecordsCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func (db *DB) GetGeneratedRecord(ctx context.Context, userID, recordID string) (*GeneratedRecord, error) {
	var r GeneratedRecord
	err := db.Pool.QueryRow(ctx, 
		"SELECT id, user_id, template_name, file_path, records_count, created_at FROM generated_records WHERE user_id = $1 AND id = $2", userID, recordID).Scan(
		&r.ID, &r.UserID, &r.TemplateName, &r.FilePath, &r.RecordsCount, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get generated record: %w", err)
	}
	return &r, nil
}
