package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"libraryhub/apps/api/internal/auth"
	"libraryhub/apps/api/internal/books"
)

// Handler は認証関連エンドポイントを提供する HTTP ハンドラー。
type Handler struct {
	authService authService
	bookService bookService
}

type authService interface {
	Login(ctx context.Context, in auth.LoginInput) (auth.LoginResult, error)
	ParseToken(token string) (auth.Claims, error)
	EnsureUserActive(ctx context.Context, userID string) error
	UpdateUserStatus(ctx context.Context, in auth.UpdateUserStatusInput) (auth.UpdateUserStatusResult, error)
	ListAdminUsers(ctx context.Context) ([]auth.AdminUser, error)
}

type bookService interface {
	CreateBook(ctx context.Context, in books.CreateBookInput) (books.Book, error)
	ListBooks(ctx context.Context) ([]books.Book, error)
	GetBook(ctx context.Context, bookID string) (books.Book, error)
	UpdateBook(ctx context.Context, in books.UpdateBookInput) (books.Book, error)
	DeleteBook(ctx context.Context, bookID string) error
}

type contextKey string

const authClaimsContextKey contextKey = "authClaims"

// NewHandler は HTTP ルーティングで利用するハンドラーを生成する。
func NewHandler(authService authService) *Handler {
	return NewHandlerWithBooks(authService, noopBookService{})
}

// NewHandlerWithBooks は auth/book サービスを注入したハンドラーを生成する。
func NewHandlerWithBooks(authService authService, bookService bookService) *Handler {
	return &Handler{
		authService: authService,
		bookService: bookService,
	}
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

	r.Route("/admin/books", func(r chi.Router) {
		// 蔵書 CRUD は管理者操作のみ許可する。
		r.Use(h.requireAuth)
		r.Use(h.requireRoles("ADMIN"))
		r.Post("/", h.createBook)
		r.Get("/", h.listBooks)
		r.Get("/{bookId}", h.getBook)
		r.Patch("/{bookId}", h.updateBook)
		r.Delete("/{bookId}", h.deleteBook)
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

type bookRequest struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	ISBN          *string `json:"isbn"`
	Publisher     *string `json:"publisher"`
	PublishedYear *int    `json:"published_year"`
	CategoryID    string  `json:"category_id"`
	Location      *string `json:"location"`
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
	targetUserID := chi.URLParam(r, "userId")
	if isSameUserID(claims.UserID, targetUserID) {
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
		UserID: targetUserID,
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

// createBook は管理者による蔵書新規登録を受け付ける。
func (h *Handler) createBook(w http.ResponseWriter, r *http.Request) {
	var req bookRequest
	// 入力 JSON が壊れている場合は業務処理に進めない。
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	book, err := h.bookService.CreateBook(r.Context(), books.CreateBookInput{
		Title:         req.Title,
		Author:        req.Author,
		ISBN:          req.ISBN,
		Publisher:     req.Publisher,
		PublishedYear: req.PublishedYear,
		CategoryID:    req.CategoryID,
		Location:      req.Location,
	})
	if err != nil {
		switch err {
		case books.ErrInvalidBookInput:
			writeError(w, http.StatusBadRequest, "invalid book input")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"book": book})
}

// listBooks は管理者向け蔵書一覧を返す。
func (h *Handler) listBooks(w http.ResponseWriter, r *http.Request) {
	items, err := h.bookService.ListBooks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"books": items})
}

// getBook は管理者向けの蔵書詳細を返す。
func (h *Handler) getBook(w http.ResponseWriter, r *http.Request) {
	bookID := chi.URLParam(r, "bookId")

	book, err := h.bookService.GetBook(r.Context(), bookID)
	if err != nil {
		switch err {
		case books.ErrInvalidBookID:
			writeError(w, http.StatusBadRequest, "invalid book id")
		case books.ErrBookNotFound:
			writeError(w, http.StatusNotFound, "book not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"book": book})
}

// updateBook は管理者による蔵書編集を受け付ける。
func (h *Handler) updateBook(w http.ResponseWriter, r *http.Request) {
	bookID := chi.URLParam(r, "bookId")

	var req bookRequest
	// 入力 JSON が壊れている場合は業務処理に進めない。
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	book, err := h.bookService.UpdateBook(r.Context(), books.UpdateBookInput{
		BookID:        bookID,
		Title:         req.Title,
		Author:        req.Author,
		ISBN:          req.ISBN,
		Publisher:     req.Publisher,
		PublishedYear: req.PublishedYear,
		CategoryID:    req.CategoryID,
		Location:      req.Location,
	})
	if err != nil {
		switch err {
		case books.ErrInvalidBookID:
			writeError(w, http.StatusBadRequest, "invalid book id")
		case books.ErrInvalidBookInput:
			writeError(w, http.StatusBadRequest, "invalid book input")
		case books.ErrBookNotFound:
			writeError(w, http.StatusNotFound, "book not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"book": book})
}

// deleteBook は管理者による蔵書削除を実行する。
func (h *Handler) deleteBook(w http.ResponseWriter, r *http.Request) {
	bookID := chi.URLParam(r, "bookId")

	err := h.bookService.DeleteBook(r.Context(), bookID)
	if err != nil {
		switch err {
		case books.ErrInvalidBookID:
			writeError(w, http.StatusBadRequest, "invalid book id")
		case books.ErrBookNotFound:
			writeError(w, http.StatusNotFound, "book not found")
		case books.ErrBookDeleteRestricted:
			writeError(w, http.StatusConflict, "book cannot be deleted due to related histories")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isSameUserID は UUID の表記ゆれを吸収して同一ユーザーかを判定する。
func isSameUserID(left, right string) bool {
	normalizedLeft, okLeft := normalizeUUID(left)
	normalizedRight, okRight := normalizeUUID(right)
	if !okLeft || !okRight {
		return false
	}
	return normalizedLeft == normalizedRight
}

// normalizeUUID は UUID を比較しやすい 32 桁の16進文字列へ正規化する。
func normalizeUUID(value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	normalized = strings.TrimPrefix(strings.ToLower(normalized), "urn:uuid:")
	normalized = strings.TrimPrefix(normalized, "{")
	normalized = strings.TrimSuffix(normalized, "}")
	normalized = strings.ReplaceAll(normalized, "-", "")
	if len(normalized) != 32 {
		return "", false
	}
	for _, char := range normalized {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return "", false
		}
	}
	return normalized, true
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
		if err := h.authService.EnsureUserActive(r.Context(), claims.UserID); err != nil {
			switch err {
			case auth.ErrUserInactive, auth.ErrUserNotFound:
				writeError(w, http.StatusUnauthorized, "invalid token")
			default:
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
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

type noopBookService struct{}

func (noopBookService) CreateBook(context.Context, books.CreateBookInput) (books.Book, error) {
	return books.Book{}, books.ErrInvalidBookInput
}

func (noopBookService) ListBooks(context.Context) ([]books.Book, error) {
	return []books.Book{}, nil
}

func (noopBookService) GetBook(context.Context, string) (books.Book, error) {
	return books.Book{}, books.ErrBookNotFound
}

func (noopBookService) UpdateBook(context.Context, books.UpdateBookInput) (books.Book, error) {
	return books.Book{}, books.ErrBookNotFound
}

func (noopBookService) DeleteBook(context.Context, string) error {
	return books.ErrBookNotFound
}
