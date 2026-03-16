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

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidCredentials は識別子またはパスワードが不正なときに返す。
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidToken は JWT の形式・署名・有効期限が不正なときに返す。
var ErrInvalidToken = errors.New("invalid token")

// Service はログイン認証とトークン発行を担当する。
type Service struct {
	db           *pgxpool.Pool
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
	PasswordHash string
}

// Claims は認証済みリクエストで利用する JWT クレーム。
type Claims struct {
	UserID    string
	Role      string
	ExpiresAt time.Time
}

// NewService は認証サービスを生成する。
func NewService(db *pgxpool.Pool, jwtSecret string, jwtExpiresIn time.Duration) *Service {
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
		SELECT id::text, employee_id, email, role::text, password_hash
		FROM users
		WHERE status = 'ACTIVE'
		  AND (email = $1 OR employee_id = $1)
		LIMIT 1
	`, identifier).Scan(
		&user.ID,
		&user.EmployeeID,
		&user.Email,
		&user.Role,
		&user.PasswordHash,
	)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	// ユーザー保存済みのハッシュと入力パスワードを照合する。
	if !VerifyPasswordHash(user.PasswordHash, in.Password) {
		return LoginResult{}, ErrInvalidCredentials
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
