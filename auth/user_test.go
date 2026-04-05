package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestHashPassword_ReturnsValidBcryptHash: result must be a valid bcrypt hash
func TestHashPassword_ReturnsValidBcryptHash(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("expected non-empty hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret")); err != nil {
		t.Errorf("hash is not a valid bcrypt hash: %v", err)
	}
}

// TestHashPassword_DifferentFromPlainText: hash must not equal the plain text input
func TestHashPassword_DifferentFromPlainText(t *testing.T) {
	plain := "secret"
	hash, _ := HashPassword(plain)
	if hash == plain {
		t.Error("hash should not equal plain text")
	}
}

// TestHashPassword_UniquePerCall: two calls with the same input must produce different hashes (bcrypt salt)
func TestHashPassword_UniquePerCall(t *testing.T) {
	hash1, _ := HashPassword("secret")
	hash2, _ := HashPassword("secret")
	if hash1 == hash2 {
		t.Error("expected different hashes for same input due to bcrypt salt")
	}
}

// TestCheckPassword_CorrectPassword: matching password returns true
func TestCheckPassword_CorrectPassword(t *testing.T) {
	hash, _ := HashPassword("correct")
	if !CheckPassword("correct", hash) {
		t.Error("expected true for correct password")
	}
}

// TestCheckPassword_WrongPassword: wrong password returns false
func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct")
	if CheckPassword("wrong", hash) {
		t.Error("expected false for wrong password")
	}
}

// TestCheckPassword_EmptyPassword: empty password returns false
func TestCheckPassword_EmptyPassword(t *testing.T) {
	hash, _ := HashPassword("correct")
	if CheckPassword("", hash) {
		t.Error("expected false for empty password")
	}
}

// TestCheckPassword_InvalidHash: malformed hash string returns false without panicking
func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("secret", "not-a-valid-hash") {
		t.Error("expected false for invalid hash")
	}
}
