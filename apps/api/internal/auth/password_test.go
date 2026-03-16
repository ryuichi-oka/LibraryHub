package auth

import "testing"

func TestHashPasswordAndVerifyPasswordHash(t *testing.T) {
	const raw = "libraryhub-password"

	hashed, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hashed == raw {
		t.Fatalf("hashed password must not equal raw password")
	}

	if ok := VerifyPasswordHash(hashed, raw); !ok {
		t.Fatalf("VerifyPasswordHash() = false, want true")
	}
	if ok := VerifyPasswordHash(hashed, "wrong-password"); ok {
		t.Fatalf("VerifyPasswordHash() = true with wrong password, want false")
	}
}

func TestHashPasswordRejectsInvalidInput(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatalf("HashPassword(\"\") must return error")
	}

	tooLong := make([]byte, 73)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	if _, err := HashPassword(string(tooLong)); err == nil {
		t.Fatalf("HashPassword() must reject passwords over 72 bytes")
	}
}

func TestVerifyPasswordHashRejectsInvalidInput(t *testing.T) {
	if ok := VerifyPasswordHash("", "password"); ok {
		t.Fatalf("VerifyPasswordHash() must reject empty hash")
	}
	if ok := VerifyPasswordHash("$2a$10$abcdefghijklmnopqrstuuuuuuuuuuuuuuuuuuuuuuuuuuuuuu", ""); ok {
		t.Fatalf("VerifyPasswordHash() must reject empty password")
	}

	tooLong := make([]byte, 73)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	if ok := VerifyPasswordHash("$2a$10$abcdefghijklmnopqrstuuuuuuuuuuuuuuuuuuuuuuuuuuuuuu", string(tooLong)); ok {
		t.Fatalf("VerifyPasswordHash() must reject passwords over 72 bytes")
	}
}
