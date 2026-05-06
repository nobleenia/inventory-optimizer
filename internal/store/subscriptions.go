package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Subscription struct {
	UserID               string
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               string
	CurrentPeriodEnd     time.Time
	UpdatedAt            time.Time
}

// GetSubscription fetches the subscription status for a user.
func (db *DB) GetSubscription(ctx context.Context, userID string) (*Subscription, error) {
	sub := &Subscription{UserID: userID}

	err := db.Pool.QueryRow(ctx,
		`SELECT stripe_customer_id, stripe_subscription_id, status, current_period_end, updated_at
 FROM subscriptions WHERE user_id = $1`,
		userID,
	).Scan(
		&sub.StripeCustomerID, &sub.StripeSubscriptionID,
		&sub.Status, &sub.CurrentPeriodEnd, &sub.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No subscription found, not an error.
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	return sub, nil
}

// UpsertSubscription creates or updates a subscription record.
func (db *DB) UpsertSubscription(ctx context.Context, sub *Subscription) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id, stripe_customer_id, stripe_subscription_id, status, current_period_end, updated_at)
 VALUES ($1, $2, $3, $4, $5, now())
 ON CONFLICT (user_id) DO UPDATE SET 
 stripe_customer_id = EXCLUDED.stripe_customer_id,
 stripe_subscription_id = EXCLUDED.stripe_subscription_id,
 status = EXCLUDED.status,
 current_period_end = EXCLUDED.current_period_end,
 updated_at = now()`,
		sub.UserID, sub.StripeCustomerID, sub.StripeSubscriptionID, sub.Status, sub.CurrentPeriodEnd,
	)
	if err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}
	return nil
}
