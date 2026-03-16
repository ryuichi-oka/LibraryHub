package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

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

	handler, adminToken := newHandlerWithToken(t, "ADMIN", time.Now().Add(time.Hour))
	_, userToken := newHandlerWithToken(t, "USER", time.Now().Add(time.Hour))

	router := chi.NewRouter()
	router.Use(handler.requireAuth)
	router.With(handler.requireRoles("ADMIN")).Get("/admin-only", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("admin can access", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if res.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
		}
	})

	t.Run("general user is forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if res.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
		}
	})
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
