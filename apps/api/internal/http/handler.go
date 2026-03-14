package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"libraryhub/apps/api/internal/auth"
)

// Handler は認証関連エンドポイントを提供する HTTP ハンドラー。
type Handler struct {
	authService *auth.Service
}

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
