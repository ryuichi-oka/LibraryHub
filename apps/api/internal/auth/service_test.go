package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestLoginRejectsInactiveUser(t *testing.T) {
	t.Parallel()

	hashed, err := HashPassword("Passw0rd!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	service := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{
				values: []any{"user-2", "E002", "user2@example.com", "USER", "INACTIVE", hashed},
			}
		},
	}, "libraryhub-secret", time.Hour)

	_, err = service.Login(context.Background(), LoginInput{
		Identifier: "E002",
		Password:   "Passw0rd!",
	})
	if err != ErrUserInactive {
		t.Fatalf("Login() error = %v, want %v", err, ErrUserInactive)
	}
}

func TestLoginRejectsUnknownIdentifier(t *testing.T) {
	t.Parallel()

	service := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{err: pgx.ErrNoRows}
		},
	}, "libraryhub-secret", time.Hour)

	_, err := service.Login(context.Background(), LoginInput{
		Identifier: "not-found@example.com",
		Password:   "Passw0rd!",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestLoginReturnsErrorWhenUserLookupFails(t *testing.T) {
	t.Parallel()

	service := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{err: errors.New("db unavailable")}
		},
	}, "libraryhub-secret", time.Hour)

	_, err := service.Login(context.Background(), LoginInput{
		Identifier: "E001",
		Password:   "Passw0rd!",
	})
	if err == nil {
		t.Fatalf("Login() error = nil, want non-nil")
	}
	if errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, should not be invalid credentials", err)
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

func TestListAdminUsers(t *testing.T) {
	t.Parallel()

	service := newService(fakeDB{
		queryFunc: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &fakeRows{
				values: [][]any{
					{"user-1", "E001", "user1@example.com", "ADMIN", "ACTIVE"},
					{"user-2", "E002", "user2@example.com", "USER", "INACTIVE"},
				},
			}, nil
		},
	}, "libraryhub-secret", time.Hour)

	users, err := service.ListAdminUsers(context.Background())
	if err != nil {
		t.Fatalf("ListAdminUsers() error = %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("users count = %d, want 2", len(users))
	}
	if users[0].EmployeeID != "E001" {
		t.Fatalf("users[0].EmployeeID = %q, want %q", users[0].EmployeeID, "E001")
	}
	if users[1].Status != "INACTIVE" {
		t.Fatalf("users[1].Status = %q, want %q", users[1].Status, "INACTIVE")
	}
}

type fakeDB struct {
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (db fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if db.queryRowFunc == nil {
		return fakeRow{err: errors.New("unexpected query")}
	}
	return db.queryRowFunc(ctx, sql, args...)
}

func (db fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if db.queryFunc == nil {
		return nil, errors.New("unexpected query")
	}
	return db.queryFunc(ctx, sql, args...)
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

type fakeRows struct {
	values [][]any
	index  int
}

func (r *fakeRows) Close() {}
func (r *fakeRows) Err() error {
	return nil
}
func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *fakeRows) Next() bool {
	return r.index < len(r.values)
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.index >= len(r.values) {
		return errors.New("no rows")
	}
	row := r.values[r.index]
	r.index++

	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = row[i].(string)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}
func (r *fakeRows) Values() ([]any, error) {
	return nil, errors.New("not implemented")
}
func (r *fakeRows) RawValues() [][]byte {
	return nil
}
func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}
