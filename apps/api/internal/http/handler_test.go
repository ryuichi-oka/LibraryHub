package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"libraryhub/apps/api/internal/auth"
)

func TestRequireAuthAllowsAccessToMe(t *testing.T) {
	t.Parallel()

	handler, token := newHandlerWithToken(t, "USER", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body struct {
		User struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.User.ID != "user-1" {
		t.Fatalf("user.id = %q, want %q", body.User.ID, "user-1")
	}
	if body.User.Role != "USER" {
		t.Fatalf("user.role = %q, want %q", body.User.Role, "USER")
	}
}

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	t.Parallel()

	handler, _ := newHandlerWithToken(t, "USER", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestRequireRoles(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeAuthService{
		parseTokenFunc: func(token string) (auth.Claims, error) {
			switch token {
			case "admin-token":
				return auth.Claims{UserID: "admin-1", Role: "ADMIN", ExpiresAt: time.Now().Add(time.Hour)}, nil
			case "user-token":
				return auth.Claims{UserID: "user-1", Role: "USER", ExpiresAt: time.Now().Add(time.Hour)}, nil
			default:
				return auth.Claims{}, auth.ErrInvalidToken
			}
		},
		updateUserStatusFunc: func(_ context.Context, _ auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error) {
			return auth.UpdateUserStatusResult{
				UserID:    "user-2",
				Status:    "INACTIVE",
				UpdatedAt: time.Now().Format(time.RFC3339),
			}, nil
		},
	})

	t.Run("admin can access", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/users/user-2/status", strings.NewReader(`{"status":"INACTIVE"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer admin-token")
		res := httptest.NewRecorder()

		handler.Routes().ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
		}
	})

	t.Run("general user is forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/users/user-2/status", strings.NewReader(`{"status":"INACTIVE"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer user-token")
		res := httptest.NewRecorder()

		handler.Routes().ServeHTTP(res, req)

		if res.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
		}
	})
}

func TestUpdateUserStatus(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC)
	service := fakeAuthService{
		parseTokenFunc: func(token string) (auth.Claims, error) {
			return auth.Claims{
				UserID:    "admin-1",
				Role:      "ADMIN",
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		updateUserStatusFunc: func(_ context.Context, in auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error) {
			if got, want := in.UserID, "user-2"; got != want {
				t.Fatalf("user id = %v, want %v", got, want)
			}
			if got, want := in.Status, "inactive"; got != want {
				t.Fatalf("status = %v, want %v", got, want)
			}
			return auth.UpdateUserStatusResult{
				UserID:    "user-2",
				Status:    "INACTIVE",
				UpdatedAt: updatedAt.Format(time.RFC3339),
			}, nil
		},
	}

	handler := NewHandler(service)
	token, err := authToken("libraryhub-secret", "admin-1", "ADMIN", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-2/status", strings.NewReader(`{"status":"inactive"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body struct {
		User struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			UpdatedAt string `json:"updated_at"`
		} `json:"user"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.User.ID != "user-2" {
		t.Fatalf("user.id = %q, want %q", body.User.ID, "user-2")
	}
	if body.User.Status != "INACTIVE" {
		t.Fatalf("user.status = %q, want %q", body.User.Status, "INACTIVE")
	}
	if body.User.UpdatedAt != updatedAt.Format(time.RFC3339) {
		t.Fatalf("user.updated_at = %q, want %q", body.User.UpdatedAt, updatedAt.Format(time.RFC3339))
	}
}

func TestUpdateUserStatusRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeAuthService{
		updateUserStatusFunc: func(_ context.Context, _ auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error) {
			return auth.UpdateUserStatusResult{}, auth.ErrInvalidUserStatus
		},
		parseTokenFunc: func(token string) (auth.Claims, error) {
			return auth.Claims{
				UserID:    "admin-1",
				Role:      "ADMIN",
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	})
	_, token := newHandlerWithToken(t, "ADMIN", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodPost, "/admin/users/user-2/status", strings.NewReader(`{"status":"disabled"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func newHandlerWithToken(t *testing.T, role string, expiresAt time.Time) (*Handler, string) {
	t.Helper()

	service := auth.NewService(nil, "libraryhub-secret", time.Hour)
	token, err := authToken("libraryhub-secret", "user-1", role, expiresAt)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	return NewHandler(service), token
}

func authToken(secret, userID, role string, expiresAt time.Time) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"sub":  userID,
		"role": role,
		"exp":  expiresAt.Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", err
	}

	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature, nil
}

type fakeAuthService struct {
	loginFunc            func(ctx context.Context, in auth.LoginInput) (auth.LoginResult, error)
	parseTokenFunc       func(token string) (auth.Claims, error)
	updateUserStatusFunc func(ctx context.Context, in auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error)
}

func (s fakeAuthService) Login(ctx context.Context, in auth.LoginInput) (auth.LoginResult, error) {
	if s.loginFunc == nil {
		return auth.LoginResult{}, errors.New("unexpected login")
	}
	return s.loginFunc(ctx, in)
}

func (s fakeAuthService) ParseToken(token string) (auth.Claims, error) {
	if s.parseTokenFunc == nil {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	return s.parseTokenFunc(token)
}

func (s fakeAuthService) UpdateUserStatus(ctx context.Context, in auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error) {
	if s.updateUserStatusFunc == nil {
		return auth.UpdateUserStatusResult{}, errors.New("unexpected update user status")
	}
	return s.updateUserStatusFunc(ctx, in)
}
