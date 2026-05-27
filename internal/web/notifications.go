package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/store"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type noopMailer struct{}

func (noopMailer) Send(context.Context, string, string, string) error { return nil }

type logMailer struct{}

func (logMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("notification email to %s: %s\n%s", to, subject, body)
	return nil
}

type smtpMailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func newMailerFromEnv() Mailer {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return logMailer{}
	}

	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	}
	if from == "" {
		from = "inventory-optimizer@localhost"
	}

	return &smtpMailer{
		host:     host,
		port:     port,
		username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		from:     from,
	}
}

func (m *smtpMailer) Send(_ context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(m.host, m.port)
	message := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	var auth smtp.Auth
	if m.username != "" || m.password != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(message))
}

type replenishmentAlert struct {
	SKU                 string
	Shortage            int
	CurrentInventory    int
	ReorderPoint        int
	ProjectedAnnualCost float64
}

func buildReplenishmentAlerts(results []models.SKUReport, limit int) []replenishmentAlert {
	alerts := make([]replenishmentAlert, 0)
	for _, result := range results {
		shortage := int(result.Policy.ReorderPoint - float64(result.Parameters.CurrentInventory))
		if shortage <= 0 {
			continue
		}
		alerts = append(alerts, replenishmentAlert{
			SKU:                 result.Parameters.SKU,
			Shortage:            shortage,
			CurrentInventory:    result.Parameters.CurrentInventory,
			ReorderPoint:        int(result.Policy.ReorderPoint),
			ProjectedAnnualCost: result.Simulation.AvgTotalAnnualCost,
		})
	}

	if len(alerts) > 1 {
		for i := 0; i < len(alerts); i++ {
			for j := i + 1; j < len(alerts); j++ {
				if alerts[j].Shortage > alerts[i].Shortage {
					alerts[i], alerts[j] = alerts[j], alerts[i]
				}
			}
		}
	}
	if limit > 0 && len(alerts) > limit {
		alerts = alerts[:limit]
	}
	return alerts
}

func (s *Server) enqueueReplenishmentNotifications(ctx context.Context, userID, reportID string, results []models.SKUReport) error {
	if s.db == nil {
		return nil
	}
	alerts := buildReplenishmentAlerts(results, 3)
	for _, alert := range alerts {
		notification := &store.Notification{
			UserID:   userID,
			Kind:     "replenishment",
			Title:    fmt.Sprintf("Replenish SKU %s", alert.SKU),
			Body:     fmt.Sprintf("SKU %s is %d units below reorder point. Stock %d vs ROP %d. Projected annual cost €%.2f.", alert.SKU, alert.Shortage, alert.CurrentInventory, alert.ReorderPoint, alert.ProjectedAnnualCost),
			ReportID: reportID,
		}
		if err := s.db.CreateNotification(ctx, notification); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) startNotificationWorker(ctx context.Context) {
	if s.db == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		if err := s.dispatchNotificationEmails(ctx); err != nil {
			log.Printf("notification dispatch failed: %v", err)
		}
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := s.dispatchNotificationEmails(ctx); err != nil {
				log.Printf("notification dispatch failed: %v", err)
			}
		}
	}()
}

func (s *Server) dispatchNotificationEmails(ctx context.Context) error {
	if s.db == nil {
		return nil
	}

	settings, err := s.db.ListNotificationSettings(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, setting := range settings {
		if !setting.Enabled {
			continue
		}
		if !s.hasPremiumAccessForUser(ctx, setting.UserID) {
			continue
		}
		if !isNotificationScheduleDue(setting, now) {
			continue
		}

		notifications, err := s.db.ListNotifications(ctx, setting.UserID, 25)
		if err != nil {
			log.Printf("notification email load failed for %s: %v", setting.UserID, err)
			continue
		}

		pending := filterUnsentNotifications(notifications, setting.LastSentAt)
		if len(pending) == 0 {
			continue
		}

		recipient := strings.TrimSpace(setting.EmailOverride)
		if recipient == "" {
			user, err := s.db.GetUserByID(ctx, setting.UserID)
			if err != nil {
				log.Printf("notification email user lookup failed for %s: %v", setting.UserID, err)
				continue
			}
			recipient = user.Email
		}

		subject, body := composeNotificationDigest(pending)
		if err := s.mailer.Send(ctx, recipient, subject, body); err != nil {
			log.Printf("notification email send failed for %s: %v", recipient, err)
			continue
		}

		nowCopy := now
		settingsCopy := setting
		settingsCopy.LastSentAt = &nowCopy
		if err := s.db.UpsertNotificationSettings(ctx, &settingsCopy); err != nil {
			log.Printf("notification email timestamp update failed for %s: %v", setting.UserID, err)
		}
	}

	return nil
}

func isNotificationScheduleDue(setting store.NotificationSettings, now time.Time) bool {
	hour, minute := 9, 0
	if setting.ScheduledTime != "" {
		parts := strings.Split(setting.ScheduledTime, ":")
		if len(parts) == 2 {
			if parsedHour, err := strconv.Atoi(parts[0]); err == nil {
				hour = parsedHour
			}
			if parsedMinute, err := strconv.Atoi(parts[1]); err == nil {
				minute = parsedMinute
			}
		}
	}
	window := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(window) {
		return false
	}
	if setting.LastSentAt == nil {
		return true
	}

	last := setting.LastSentAt.In(now.Location())
	switch setting.Frequency {
	case "weekly":
		return now.Sub(last) >= 7*24*time.Hour
	default:
		return last.Before(window)
	}
}

func filterUnsentNotifications(notifications []store.Notification, lastSentAt *time.Time) []store.Notification {
	if len(notifications) == 0 {
		return nil
	}
	pending := make([]store.Notification, 0, len(notifications))
	for _, notification := range notifications {
		if lastSentAt != nil && !notification.CreatedAt.After(*lastSentAt) {
			continue
		}
		pending = append(pending, notification)
	}
	if len(pending) > 1 {
		for i := 0; i < len(pending); i++ {
			for j := i + 1; j < len(pending); j++ {
				if pending[j].CreatedAt.After(pending[i].CreatedAt) {
					pending[i], pending[j] = pending[j], pending[i]
				}
			}
		}
	}
	return pending
}

func composeNotificationDigest(notifications []store.Notification) (string, string) {
	subject := fmt.Sprintf("Inventory Optimizer: %d replenishment alert%s", len(notifications), pluralSuffix(len(notifications)))
	var builder strings.Builder
	builder.WriteString("Your replenishment alerts are ready.\n\n")
	for _, notification := range notifications {
		builder.WriteString("- ")
		builder.WriteString(notification.Title)
		builder.WriteString(": ")
		builder.WriteString(notification.Body)
		builder.WriteString("\n")
	}
	builder.WriteString("\nOpen Inventory Optimizer to review the alert center and take action.")
	return subject, builder.String()
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func (s *Server) hasPremiumAccessForUser(ctx context.Context, userID string) bool {
	if s.db == nil {
		return false
	}

	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return false
	}

	trialExpiresAt := user.CreatedAt.AddDate(0, freeTrialMonths, 0)
	if sub, err := s.db.GetSubscription(ctx, userID); err == nil && sub != nil {
		switch sub.Status {
		case "active":
			return true
		case "trial":
			if sub.CurrentPeriodEnd.IsZero() {
				sub.CurrentPeriodEnd = trialExpiresAt
			}
			return time.Now().Before(sub.CurrentPeriodEnd)
		}
	}

	return time.Now().Before(trialExpiresAt)
}

func (s *Server) handleAPINotifications(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, 401, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, 503, "Database not configured")
		return
	}

	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	notifications, err := s.db.ListNotifications(r.Context(), claims.Subject, limit)
	if err != nil {
		s.sendErrorJSON(w, 500, "Failed to load notifications")
		return
	}
	unreadCount, err := s.db.CountUnreadNotifications(r.Context(), claims.Subject)
	if err != nil {
		unreadCount = 0
	}

	s.sendJSON(w, 200, map[string]interface{}{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

func (s *Server) handleAPIMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, 401, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, 503, "Database not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		s.sendErrorJSON(w, 400, "Missing notification ID")
		return
	}
	if err := s.db.MarkNotificationRead(r.Context(), claims.Subject, id); err != nil {
		s.sendErrorJSON(w, 404, err.Error())
		return
	}

	s.sendJSON(w, 200, map[string]string{"message": "Notification updated"})
}

func (s *Server) handleAPINotificationSettings(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, 401, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, 503, "Database not configured")
		return
	}

	settings, err := s.db.GetNotificationSettings(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, 500, "Failed to load notification settings")
		return
	}

	s.sendJSON(w, 200, settings)
}

func (s *Server) handleAPIUpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	claims := s.currentUser(r)
	if claims == nil {
		s.sendErrorJSON(w, 401, "Unauthorized")
		return
	}
	if s.db == nil {
		s.sendErrorJSON(w, 503, "Database not configured")
		return
	}

	var payload struct {
		Enabled       bool   `json:"enabled"`
		Frequency     string `json:"frequency"`
		ScheduledTime string `json:"scheduled_time"`
		EmailOverride string `json:"email_override"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendErrorJSON(w, 400, "Invalid JSON payload")
		return
	}

	frequency := strings.ToLower(strings.TrimSpace(payload.Frequency))
	if frequency == "" {
		frequency = "daily"
	}
	if frequency != "daily" && frequency != "weekly" {
		s.sendErrorJSON(w, 400, "Frequency must be daily or weekly")
		return
	}

	scheduledTime := strings.TrimSpace(payload.ScheduledTime)
	if scheduledTime == "" {
		scheduledTime = "09:00"
	}
	if _, err := time.Parse("15:04", scheduledTime); err != nil {
		s.sendErrorJSON(w, 400, "Scheduled time must use HH:MM")
		return
	}

	settings, err := s.db.GetNotificationSettings(r.Context(), claims.Subject)
	if err != nil {
		s.sendErrorJSON(w, 500, "Failed to load notification settings")
		return
	}
	settings.Enabled = payload.Enabled
	settings.Frequency = frequency
	settings.ScheduledTime = scheduledTime
	settings.EmailOverride = strings.TrimSpace(payload.EmailOverride)
	settings.Timezone = "UTC"

	if err := s.db.UpsertNotificationSettings(r.Context(), settings); err != nil {
		s.sendErrorJSON(w, 500, "Failed to save notification settings")
		return
	}

	s.sendJSON(w, 200, settings)
}
