package auth

import (
	"testing"
	"time"
)

func testConfig() Config {
	return Config{
		Secret:          "test-secret-key-at-least-32-chars-long",
		Issuer:          "test",
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" || hash == "mypassword" {
		t.Fatal("expected a bcrypt hash, not plaintext")
	}
}

func TestCheckPassword_Valid(t *testing.T) {
	hash, _ := HashPassword("correcthorse")
	if err := CheckPassword(hash, "correcthorse"); err != nil {
		t.Fatalf("expected nil error for correct password, got: %v", err)
	}
}

func TestCheckPassword_Invalid(t *testing.T) {
	hash, _ := HashPassword("correcthorse")
	err := CheckPassword(hash, "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestGenerateTokenPair(t *testing.T) {
	svc := NewService(testConfig())
	pair, err := svc.GenerateTokenPair("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("access token is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("refresh token is empty")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Error("access and refresh tokens should be different")
	}
	if pair.ExpiresIn != 300 {
		t.Errorf("expected ExpiresIn=300, got %d", pair.ExpiresIn)
	}
}

func TestValidateAccessToken(t *testing.T) {
	svc := NewService(testConfig())
	pair, _ := svc.GenerateTokenPair("user-456", "user@example.com")

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.UserID != "user-456" {
		t.Errorf("expected UserID=user-456, got %s", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("expected Email=user@example.com, got %s", claims.Email)
	}
	if claims.Type != "access" {
		t.Errorf("expected Type=access, got %s", claims.Type)
	}
}

func TestValidateRefreshToken(t *testing.T) {
	svc := NewService(testConfig())
	pair, _ := svc.GenerateTokenPair("user-789", "refresh@example.com")

	claims, err := svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
	if claims.UserID != "user-789" {
		t.Errorf("expected UserID=user-789, got %s", claims.UserID)
	}
	if claims.Type != "refresh" {
		t.Errorf("expected Type=refresh, got %s", claims.Type)
	}
}

func TestAccessTokenRejectsRefresh(t *testing.T) {
	svc := NewService(testConfig())
	pair, _ := svc.GenerateTokenPair("u1", "a@b.com")

	// A refresh token should not pass access token validation.
	_, err := svc.ValidateAccessToken(pair.RefreshToken)
	if err == nil {
		t.Fatal("expected error when validating refresh token as access token")
	}
}

func TestRefreshTokenRejectsAccess(t *testing.T) {
	svc := NewService(testConfig())
	pair, _ := svc.GenerateTokenPair("u1", "a@b.com")

	_, err := svc.ValidateRefreshToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error when validating access token as refresh token")
	}
}

func TestInvalidTokenString(t *testing.T) {
	svc := NewService(testConfig())
	_, err := svc.ValidateAccessToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}

func TestWrongSecret(t *testing.T) {
	svc1 := NewService(Config{
		Secret:          "secret-one",
		Issuer:          "test",
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
	})
	svc2 := NewService(Config{
		Secret:          "secret-two",
		Issuer:          "test",
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
	})

	pair, _ := svc1.GenerateTokenPair("u1", "a@b.com")
	_, err := svc2.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error when validating with wrong secret")
	}
}

func TestExpiredToken(t *testing.T) {
	cfg := Config{
		Secret:          "test-secret",
		Issuer:          "test",
		AccessTokenTTL:  -1 * time.Second, // already expired
		RefreshTokenTTL: 1 * time.Hour,
	}
	svc := NewService(cfg)
	pair, _ := svc.GenerateTokenPair("u1", "a@b.com")

	_, err := svc.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("my-secret")
	if cfg.Secret != "my-secret" {
		t.Errorf("expected secret=my-secret, got %s", cfg.Secret)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("expected 15m access TTL, got %v", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("expected 7d refresh TTL, got %v", cfg.RefreshTokenTTL)
	}
}
