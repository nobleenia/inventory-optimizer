// Package auth provides JWT token creation/validation and password
// hashing using bcrypt. It is consumed by the API layer and never
// touches HTTP directly.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidToken is returned when a JWT cannot be parsed or is expired.
var ErrInvalidToken = errors.New("invalid or expired token")

// ErrInvalidCredentials is returned when login credentials don't match.
var ErrInvalidCredentials = errors.New("invalid email or password")

// Config holds JWT signing parameters.
type Config struct {
	Secret          string        // HMAC-SHA256 signing key
	Issuer          string        // "iss" claim
	AccessTokenTTL  time.Duration // lifetime of access tokens
	RefreshTokenTTL time.Duration // lifetime of refresh tokens
}

// DefaultConfig returns sensible defaults for the REST API (short-lived access tokens).
func DefaultConfig(secret string) Config {
	return Config{
		Secret:          secret,
		Issuer:          "inventory-optimizer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
}

// WebConfig returns config suitable for browser sessions (longer-lived access tokens).
func WebConfig(secret string) Config {
	return Config{
		Secret:          secret,
		Issuer:          "inventory-optimizer",
		AccessTokenTTL:  7 * 24 * time.Hour, // browser cookie session
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}
}

// Claims contains the JWT payload.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair holds an access token and a refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds until access token expires
}

// Service provides auth operations.
type Service struct {
	cfg Config
}

// NewService creates an auth service with the given config.
func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// HashPassword returns a bcrypt hash of the plaintext password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

// GenerateTokenPair creates a new access + refresh token pair.
func (s *Service) GenerateTokenPair(userID, email string) (*TokenPair, error) {
	now := time.Now()

	accessToken, err := s.createToken(userID, email, "access", now, s.cfg.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("create access token: %w", err)
	}

	refreshToken, err := s.createToken(userID, email, "refresh", now, s.cfg.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// ValidateAccessToken parses and validates a JWT access token.
func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	claims, err := s.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != "access" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ValidateRefreshToken parses and validates a JWT refresh token.
func (s *Service) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	claims, err := s.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (s *Service) createToken(userID, email, tokenType string, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

func (s *Service) parseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
