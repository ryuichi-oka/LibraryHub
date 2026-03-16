package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"libraryhub/apps/api/internal/auth"
)

// Handler は認証関連エンドポイントを提供する HTTP ハンドラー。
type Handler struct {
	authService *auth.Service
}

type contextKey string

const authClaimsContextKey contextKey = "authClaims"

// NewHandler は HTTP ルーティングで利用するハンドラーを生成する。
func NewHandler(authService *auth.Service) *Handler {
	return &Handler{authService: authService}
}

// Routes は API エンドポイントのルーティング定義を返す。
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/auth", func(r chi.Router) {
		// MVP の認証入口。メール/社員ID + パスワードでログインする。
		r.Post("/login", h.login)
	})

	r.Group(func(r chi.Router) {
		// 認証済みユーザーだけが自分の情報を取得できる。
		r.Use(h.requireAuth)
		r.Get("/me", h.me)
	})

	return r
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	// 入力 JSON が不正な場合は即時に 400 を返す。
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.Login(r.Context(), auth.LoginInput{
		Identifier: req.Identifier,
		Password:   req.Password,
	})
	if err != nil {
		// 認証失敗は 401、システムエラーは 500 として区別する。
		if err == auth.ErrInvalidCredentials {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token_type": "Bearer",
		"token":      result.Token,
		"expires_at": result.ExpiresAt,
		"user": map[string]string{
			"id":          result.UserID,
			"employee_id": result.EmployeeID,
			"email":       result.Email,
			"role":        result.Role,
		},
	})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// MVP では token から取り出した最小限の利用者情報だけを返す。
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id":   claims.UserID,
			"role": claims.Role,
		},
		"expires_at": claims.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization: Bearer <token> の形式だけを受け付ける。
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		token, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		claims, err := h.authService.ParseToken(strings.TrimSpace(token))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// 後続ハンドラーが role/user_id を参照できるように context へ載せる。
		ctx := r.Context()
		ctx = contextWithAuthClaims(ctx, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) requireRoles(roles ...string) func(http.Handler) http.Handler {
	// 呼び出し時に許可ロール集合へ変換して、各リクエストで再利用する。
	allowedRoles := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowedRoles[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := authClaimsFromContext(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			// 管理者専用 API などで role を見て入口で拒否する。
			if _, allowed := allowedRoles[claims.Role]; !allowed {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func contextWithAuthClaims(ctx context.Context, claims auth.Claims) context.Context {
	// request 単位で認証済みユーザー情報を受け渡す。
	return context.WithValue(ctx, authClaimsContextKey, claims)
}

func authClaimsFromContext(r *http.Request) (auth.Claims, bool) {
	// requireAuth が設定した認証情報を後続処理から取り出す。
	claims, ok := r.Context().Value(authClaimsContextKey).(auth.Claims)
	return claims, ok
}

// writeError は API 共通のエラーレスポンスを返す。
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeJSON は API 共通の JSON レスポンスを書き込む。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
