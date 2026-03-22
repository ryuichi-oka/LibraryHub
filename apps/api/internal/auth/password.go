package auth

import "golang.org/x/crypto/bcrypt"

const bcryptMaxPasswordLength = 72

// HashPassword は平文パスワードを bcrypt ハッシュへ変換する。
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrInvalidCredentials
	}
	if len(password) > bcryptMaxPasswordLength {
		return "", ErrInvalidCredentials
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPasswordHash は保存済みハッシュと入力パスワードが一致するかを検証する。
func VerifyPasswordHash(storedHash, password string) bool {
	if storedHash == "" || password == "" {
		return false
	}
	if len(password) > bcryptMaxPasswordLength {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}
