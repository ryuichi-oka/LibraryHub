package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidCredentials は識別子またはパスワードが不正なときに返す。
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserInactive はユーザーが無効状態のときに返す。
var ErrUserInactive = errors.New("user inactive")

// ErrInvalidToken は JWT の形式・署名・有効期限が不正なときに返す。
var ErrInvalidToken = errors.New("invalid token")

// ErrInvalidUserStatus は許可されていないユーザー状態が指定されたときに返す。
var ErrInvalidUserStatus = errors.New("invalid user status")

// ErrUserNotFound は対象ユーザーが見つからないときに返す。
var ErrUserNotFound = errors.New("user not found")

type dbQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Service はログイン認証とトークン発行を担当する。
type Service struct {
	db           dbQuerier
	jwtSecret    []byte
	jwtExpiresIn time.Duration
}

// LoginInput はログイン要求の入力値。
type LoginInput struct {
	Identifier string
	Password   string
}

// LoginResult はログイン成功時に呼び出し元へ返す認証結果。
type LoginResult struct {
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	Token      string `json:"token"`
	ExpiresAt  string `json:"expires_at"`
	EmployeeID string `json:"employee_id"`
	Email      string `json:"email"`
}

type userRecord struct {
	ID           string
	EmployeeID   string
	Email        string
	Role         string
	Status       string
	PasswordHash string
}

// Claims は認証済みリクエストで利用する JWT クレーム。
type Claims struct {
	UserID    string
	Role      string
	ExpiresAt time.Time
}

// UpdateUserStatusInput は管理者による利用者状態切り替えの入力値。
type UpdateUserStatusInput struct {
	UserID string
	Status string
}

// UpdateUserStatusResult は状態更新後に返す利用者情報。
type UpdateUserStatusResult struct {
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// AdminUser は管理者向けユーザー一覧の1件分を表す。
type AdminUser struct {
	UserID     string `json:"user_id"`
	EmployeeID string `json:"employee_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Status     string `json:"status"`
}

// NewService は認証サービスを生成する。
func NewService(db *pgxpool.Pool, jwtSecret string, jwtExpiresIn time.Duration) *Service {
	return newService(db, jwtSecret, jwtExpiresIn)
}

// newService はテスト差し替え可能な DB 依存で認証サービスを生成する。
func newService(db dbQuerier, jwtSecret string, jwtExpiresIn time.Duration) *Service {
	return &Service{
		db:           db,
		jwtSecret:    []byte(jwtSecret),
		jwtExpiresIn: jwtExpiresIn,
	}
}

// Login は identifier（メール/社員ID）とパスワードを検証し、JWT を発行する。
func (s *Service) Login(ctx context.Context, in LoginInput) (LoginResult, error) {
	identifier := strings.TrimSpace(in.Identifier)
	if identifier == "" || in.Password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}

	var user userRecord
	// identifier はメールアドレス/社員ID のどちらでも一致させる。
	err := s.db.QueryRow(ctx, `
		SELECT id::text, employee_id, email, role::text, status::text, password_hash
		FROM users
		WHERE email = $1 OR employee_id = $1
		LIMIT 1
	`, identifier).Scan(
		&user.ID,
		&user.EmployeeID,
		&user.Email,
		&user.Role,
		&user.Status,
		&user.PasswordHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		// DB 障害は認証失敗に丸めず、呼び出し元で 500 へ変換できるよう保持する。
		return LoginResult{}, fmt.Errorf("load user by identifier: %w", err)
	}

	// ユーザー保存済みのハッシュと入力パスワードを照合する。
	if !VerifyPasswordHash(user.PasswordHash, in.Password) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if user.Status != "ACTIVE" {
		return LoginResult{}, ErrUserInactive
	}

	now := time.Now()
	expiresAt := now.Add(s.jwtExpiresIn)
	// MVP はシンプルな HS256 署名トークンを採用する。
	token, err := buildJWT(s.jwtSecret, user.ID, user.Role, expiresAt)
	if err != nil {
		return LoginResult{}, fmt.Errorf("token build failed: %w", err)
	}

	return LoginResult{
		UserID:     user.ID,
		Role:       user.Role,
		Token:      token,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
		EmployeeID: user.EmployeeID,
		Email:      user.Email,
	}, nil
}

// ParseToken は Bearer トークンの署名と有効期限を検証する。
func (s *Service) ParseToken(token string) (Claims, error) {
	return parseJWT(s.jwtSecret, token, time.Now())
}

// EnsureUserActive は認証済みユーザーが有効状態かを確認する。
func (s *Service) EnsureUserActive(ctx context.Context, userID string) error {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return ErrUserNotFound
	}

	var status string
	err := s.db.QueryRow(ctx, `
		SELECT status::text
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`, trimmedUserID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("load user status: %w", err)
	}

	if status != "ACTIVE" {
		return ErrUserInactive
	}

	return nil
}

// UpdateUserStatus は対象ユーザーの有効/無効状態を更新する。
func (s *Service) UpdateUserStatus(ctx context.Context, in UpdateUserStatusInput) (UpdateUserStatusResult, error) {
	userID := strings.TrimSpace(in.UserID)
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	// 空値や許可外ステータスは DB 更新前に入力エラーとして扱う。
	if userID == "" || !isSupportedUserStatus(status) {
		return UpdateUserStatusResult{}, ErrInvalidUserStatus
	}

	var updatedUser struct {
		UserID    string
		Status    string
		UpdatedAt time.Time
	}

	// users.status は ACTIVE/INACTIVE だけを許可し、更新結果をそのまま返す。
	err := s.db.QueryRow(ctx, `
		UPDATE users
		SET status = $2::user_status_type,
		    updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING id::text, status::text, updated_at
	`, userID, status).Scan(
		&updatedUser.UserID,
		&updatedUser.Status,
		&updatedUser.UpdatedAt,
	)
	if err != nil {
		// MVP では詳細理由を分けず、更新不可を「対象なし」として統一する。
		return UpdateUserStatusResult{}, ErrUserNotFound
	}

	return UpdateUserStatusResult{
		UserID:    updatedUser.UserID,
		Status:    updatedUser.Status,
		UpdatedAt: updatedUser.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ListAdminUsers は管理者画面に表示するユーザー一覧を返す。
func (s *Service) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, employee_id, email, role::text, status::text
		FROM users
		ORDER BY employee_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]AdminUser, 0)
	for rows.Next() {
		var user AdminUser
		if err := rows.Scan(&user.UserID, &user.EmployeeID, &user.Email, &user.Role, &user.Status); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// buildJWT は HS256 署名付き JWT を生成する。
func buildJWT(secret []byte, userID, role string, expiresAt time.Time) (string, error) {
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

	h := base64.RawURLEncoding.EncodeToString(headerJSON)
	p := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := h + "." + p

	// header.payload へ HMAC-SHA256 署名を付与する。
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", err
	}
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + signature, nil
}

// parseJWT は JWT を分解し、署名と有効期限を検証して Claims へ変換する。
func parseJWT(secret []byte, token string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	// 署名対象は JWT の header.payload 部分のみ。
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return Claims{}, fmt.Errorf("token verify failed: %w", err)
	}

	expectedSignature := mac.Sum(nil)
	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !hmac.Equal(actualSignature, expectedSignature) {
		return Claims{}, ErrInvalidToken
	}

	// 署名検証後に payload を JSON として読み出す。
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var payload struct {
		Subject string `json:"sub"`
		Role    string `json:"role"`
		Exp     int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if payload.Subject == "" || payload.Role == "" || payload.Exp <= 0 {
		return Claims{}, ErrInvalidToken
	}

	expiresAt := time.Unix(payload.Exp, 0)
	// exp に現在時刻以上を要求し、期限切れトークンを拒否する。
	if !expiresAt.After(now) {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		UserID:    payload.Subject,
		Role:      payload.Role,
		ExpiresAt: expiresAt,
	}, nil
}

// isSupportedUserStatus は API で受け付ける利用者状態を判定する。
func isSupportedUserStatus(status string) bool {
	switch status {
	case "ACTIVE", "INACTIVE":
		return true
	default:
		return false
	}
}
