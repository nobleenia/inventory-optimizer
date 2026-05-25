package store

import (
	"context"
	"fmt"
	"time"
)

type ActivityEvent struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (db *DB) RecordActivity(ctx context.Context, event *ActivityEvent) error {
	if event == nil {
		return fmt.Errorf("activity event is required")
	}
	if event.Kind == "" {
		event.Kind = "activity"
	}
	if event.Title == "" {
		event.Title = event.Kind
	}
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO activity_events (user_id, kind, title, description, entity_type, entity_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.UserID, event.Kind, event.Title, event.Description, event.EntityType, event.EntityID,
	)
	if err != nil {
		return fmt.Errorf("record activity: %w", err)
	}
	return nil
}

func (db *DB) ListRecentActivity(ctx context.Context, userID string, limit int) ([]ActivityEvent, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, kind, title, description, entity_type, entity_id, created_at
		 FROM activity_events WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list activity: %w", err)
	}
	defer rows.Close()

	var events []ActivityEvent
	for rows.Next() {
		var event ActivityEvent
		if err := rows.Scan(&event.ID, &event.UserID, &event.Kind, &event.Title, &event.Description, &event.EntityType, &event.EntityID, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		events = append(events, event)
	}
	return events, nil
}
