package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type NotificationSettings struct {
	UserID        string     `json:"user_id"`
	Enabled       bool       `json:"enabled"`
	Frequency     string     `json:"frequency"`
	ScheduledTime string     `json:"scheduled_time"`
	EmailOverride string     `json:"email_override"`
	Timezone      string     `json:"timezone"`
	LastSentAt    *time.Time `json:"last_sent_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ReportID  string     `json:"report_id"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (db *DB) GetNotificationSettings(ctx context.Context, userID string) (*NotificationSettings, error) {
	settings := &NotificationSettings{UserID: userID}
	var lastSentAt pgtype.Timestamptz
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, enabled, frequency, scheduled_time, email_override, timezone, last_sent_at, updated_at
		 FROM notification_settings WHERE user_id = $1`,
		userID,
	).Scan(&settings.UserID, &settings.Enabled, &settings.Frequency, &settings.ScheduledTime, &settings.EmailOverride, &settings.Timezone, &lastSentAt, &settings.UpdatedAt)
	if err == pgx.ErrNoRows {
		return &NotificationSettings{
			UserID:        userID,
			Enabled:       false,
			Frequency:     "daily",
			ScheduledTime: "09:00",
			EmailOverride: "",
			Timezone:      "UTC",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification settings: %w", err)
	}
	if lastSentAt.Valid {
		t := lastSentAt.Time
		settings.LastSentAt = &t
	}
	return settings, nil
}

func (db *DB) UpsertNotificationSettings(ctx context.Context, settings *NotificationSettings) error {
	if settings == nil {
		return fmt.Errorf("notification settings are required")
	}
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO notification_settings (user_id, enabled, frequency, scheduled_time, email_override, timezone, last_sent_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		 ON CONFLICT (user_id) DO UPDATE SET
		 enabled = EXCLUDED.enabled,
		 frequency = EXCLUDED.frequency,
		 scheduled_time = EXCLUDED.scheduled_time,
		 email_override = EXCLUDED.email_override,
		 timezone = EXCLUDED.timezone,
		 last_sent_at = EXCLUDED.last_sent_at,
		 updated_at = now()`,
		settings.UserID, settings.Enabled, settings.Frequency, settings.ScheduledTime, settings.EmailOverride, settings.Timezone, settings.LastSentAt,
	)
	if err != nil {
		return fmt.Errorf("upsert notification settings: %w", err)
	}
	return nil
}

func (db *DB) ListNotificationSettings(ctx context.Context) ([]NotificationSettings, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id, enabled, frequency, scheduled_time, email_override, timezone, last_sent_at, updated_at
		 FROM notification_settings ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list notification settings: %w", err)
	}
	defer rows.Close()

	var settings []NotificationSettings
	for rows.Next() {
		var item NotificationSettings
		var lastSentAt pgtype.Timestamptz
		if err := rows.Scan(&item.UserID, &item.Enabled, &item.Frequency, &item.ScheduledTime, &item.EmailOverride, &item.Timezone, &lastSentAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan notification settings: %w", err)
		}
		if lastSentAt.Valid {
			t := lastSentAt.Time
			item.LastSentAt = &t
		}
		settings = append(settings, item)
	}
	return settings, nil
}

func (db *DB) CreateNotification(ctx context.Context, n *Notification) error {
	if n == nil {
		return fmt.Errorf("notification is required")
	}
	if n.Kind == "" {
		n.Kind = "replenishment"
	}
	if n.ReportID == "" {
		n.ReportID = ""
	}
	return db.Pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, kind, title, body, report_id, read_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		n.UserID, n.Kind, n.Title, n.Body, n.ReportID, n.ReadAt,
	).Scan(&n.ID, &n.CreatedAt)
}

func (db *DB) ListNotifications(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := db.Pool.Query(ctx,
		`SELECT id, user_id, kind, title, body, report_id, read_at, created_at
		 FROM notifications WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var item Notification
		var readAt pgtype.Timestamptz
		if err := rows.Scan(&item.ID, &item.UserID, &item.Kind, &item.Title, &item.Body, &item.ReportID, &readAt, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		if readAt.Valid {
			t := readAt.Time
			item.ReadAt = &t
		}
		notifications = append(notifications, item)
	}
	return notifications, nil
}

func (db *DB) MarkNotificationRead(ctx context.Context, userID, notificationID string) error {
	tag, err := db.Pool.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND id = $2`,
		userID, notificationID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (db *DB) CountUnreadNotifications(ctx context.Context, userID string) (int, error) {
	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		userID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}
