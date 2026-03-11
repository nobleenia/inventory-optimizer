package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// User represents a registered user.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // bcrypt hash, never serialized
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrUserNotFound is returned when a lookup finds no matching user.
var ErrUserNotFound = errors.New("user not found")

// ErrEmailTaken is returned when registration uses a duplicate email.
var ErrEmailTaken = errors.New("email already registered")

// CreateUser inserts a new user with a pre-hashed password.
func (db *DB) CreateUser(ctx context.Context, email, hashedPassword string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password)
		 VALUES ($1, $2)
		 RETURNING id, email, created_at, updated_at`,
		email, hashedPassword,
	).Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetUserByEmail retrieves a user by email address.
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, password, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID retrieves a user by UUID.
func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, password, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// isDuplicateKey checks for PostgreSQL unique violation (23505).
func isDuplicateKey(err error) bool {
	return err != nil && contains(err.Error(), "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
