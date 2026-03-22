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
	authService authService
}

type authService interface {
	Login(ctx context.Context, in auth.LoginInput) (auth.LoginResult, error)
	ParseToken(token string) (auth.Claims, error)
	UpdateUserStatus(ctx context.Context, in auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error)
	ListAdminUsers(ctx context.Context) ([]auth.AdminUser, error)
}

type contextKey string

const authClaimsContextKey contextKey = "authClaims"

// NewHandler は HTTP ルーティングで利用するハンドラーを生成する。
func NewHandler(authService authService) *Handler {
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

	r.Route("/admin/users", func(r chi.Router) {
		// ユーザー有効/無効切替は管理者だけに限定する。
		r.Use(h.requireAuth)
		r.Use(h.requireRoles("ADMIN"))
		r.Get("/", h.listAdminUsers)
		r.Post("/{userId}/status", h.updateUserStatus)
	})

	return r
}

// listAdminUsers は管理者向けのユーザー一覧を返す。
func (h *Handler) listAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.ListAdminUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]map[string]string, 0, len(users))
	for _, user := range users {
		items = append(items, map[string]string{
			"id":          user.UserID,
			"employee_id": user.EmployeeID,
			"email":       user.Email,
			"role":        user.Role,
			"status":      user.Status,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users": items,
	})
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type updateUserStatusRequest struct {
	Status string `json:"status"`
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
		if err == auth.ErrUserInactive {
			writeError(w, http.StatusForbidden, "user inactive")
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

// updateUserStatus は管理者による利用者の有効/無効切り替えを受け付ける。
func (h *Handler) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := authClaimsFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if claims.UserID == chi.URLParam(r, "userId") {
		writeError(w, http.StatusConflict, "cannot update own status")
		return
	}

	var req updateUserStatusRequest
	// 管理者操作でも JSON が壊れていれば業務処理へ進めない。
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.UpdateUserStatus(r.Context(), auth.UpdateUserStatusInput{
		UserID: chi.URLParam(r, "userId"),
		Status: req.Status,
	})
	if err != nil {
		// サービス層の業務エラーを HTTP ステータスへマッピングする。
		switch err {
		case auth.ErrInvalidUserStatus:
			writeError(w, http.StatusBadRequest, "invalid user status")
		case auth.ErrUserNotFound:
			writeError(w, http.StatusNotFound, "user not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id":         result.UserID,
			"status":     result.Status,
			"updated_at": result.UpdatedAt,
		},
	})
}

// requireAuth は Bearer トークンを検証し、認証情報を context に設定するミドルウェア。
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

// requireRoles は指定ロールだけに後続ハンドラーへのアクセスを許可する。
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

// contextWithAuthClaims は認証済みユーザー情報をリクエスト文脈へ格納する。
func contextWithAuthClaims(ctx context.Context, claims auth.Claims) context.Context {
	// request 単位で認証済みユーザー情報を受け渡す。
	return context.WithValue(ctx, authClaimsContextKey, claims)
}

// authClaimsFromContext は文脈に保存された認証情報を取り出す。
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
