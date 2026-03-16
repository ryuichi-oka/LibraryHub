package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

func TestUpdateUserStatus(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC)
	service := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, args ...any) pgx.Row {
			if got, want := args[0], "user-2"; got != want {
				t.Fatalf("user id = %v, want %v", got, want)
			}
			if got, want := args[1], "INACTIVE"; got != want {
				t.Fatalf("status = %v, want %v", got, want)
			}
			return fakeRow{
				values: []any{"user-2", "INACTIVE", updatedAt},
			}
		},
	}, "libraryhub-secret", time.Hour)

	result, err := service.UpdateUserStatus(context.Background(), UpdateUserStatusInput{
		UserID: "user-2",
		Status: "inactive",
	})
	if err != nil {
		t.Fatalf("UpdateUserStatus() error = %v", err)
	}

	if result.UserID != "user-2" {
		t.Fatalf("result.UserID = %q, want %q", result.UserID, "user-2")
	}
	if result.Status != "INACTIVE" {
		t.Fatalf("result.Status = %q, want %q", result.Status, "INACTIVE")
	}
	if result.UpdatedAt != updatedAt.Format(time.RFC3339) {
		t.Fatalf("result.UpdatedAt = %q, want %q", result.UpdatedAt, updatedAt.Format(time.RFC3339))
	}
}

func TestUpdateUserStatusRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := newService(fakeDB{}, "libraryhub-secret", time.Hour)

	_, err := service.UpdateUserStatus(context.Background(), UpdateUserStatusInput{
		UserID: "user-2",
		Status: "disabled",
	})
	if err != ErrInvalidUserStatus {
		t.Fatalf("UpdateUserStatus() error = %v, want %v", err, ErrInvalidUserStatus)
	}
}

func TestUpdateUserStatusReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{err: errors.New("no rows")}
		},
	}, "libraryhub-secret", time.Hour)

	_, err := service.UpdateUserStatus(context.Background(), UpdateUserStatusInput{
		UserID: "user-404",
		Status: "ACTIVE",
	})
	if err != ErrUserNotFound {
		t.Fatalf("UpdateUserStatus() error = %v, want %v", err, ErrUserNotFound)
	}
}

type fakeDB struct {
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (db fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if db.queryRowFunc == nil {
		return fakeRow{err: errors.New("unexpected query")}
	}
	return db.queryRowFunc(ctx, sql, args...)
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *time.Time:
			*d = r.values[i].(time.Time)
		default:
			return errors.New("unsupported scan destination")
		}
	}

	return nil
}
