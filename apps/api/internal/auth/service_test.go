package auth

import (
	"testing"
	"time"
)

func TestBuildJWTAndParseToken(t *testing.T) {
	t.Parallel()

	secret := []byte("libraryhub-secret")
	expiresAt := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	token, err := buildJWT(secret, "user-1", "ADMIN", expiresAt)
	if err != nil {
		t.Fatalf("buildJWT() error = %v", err)
	}

	claims, err := parseJWT(secret, token, expiresAt.Add(-time.Minute))
	if err != nil {
		t.Fatalf("parseJWT() error = %v", err)
	}

	if claims.UserID != "user-1" {
		t.Fatalf("claims.UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.Role != "ADMIN" {
		t.Fatalf("claims.Role = %q, want %q", claims.Role, "ADMIN")
	}
	if !claims.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("claims.ExpiresAt = %v, want %v", claims.ExpiresAt, expiresAt)
	}
}

func TestParseJWTRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	secret := []byte("libraryhub-secret")
	expiresAt := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	validToken, err := buildJWT(secret, "user-1", "USER", expiresAt)
	if err != nil {
		t.Fatalf("buildJWT() error = %v", err)
	}

	tests := map[string]string{
		"wrong format":       "not-a-jwt",
		"tampered signature": validToken + "x",
		"expired token":      validToken,
		"different secret used": func() string {
			token, _ := buildJWT([]byte("different-secret"), "user-1", "USER", expiresAt)
			return token
		}(),
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			now := expiresAt.Add(-time.Minute)
			if name == "expired token" {
				now = expiresAt
			}

			if _, err := parseJWT(secret, token, now); err != ErrInvalidToken {
				t.Fatalf("parseJWT() error = %v, want %v", err, ErrInvalidToken)
			}
		})
	}
}
