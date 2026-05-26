package auth_test

import (
	"testing"

	"spsyncapi/internal/auth"
)

func TestHashPassword_ValidInput(t *testing.T) {
	hash, err := auth.HashPassword("securePass1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := auth.HashPassword("short")
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestCheckPassword_Match(t *testing.T) {
	const plain = "correctPassword123"
	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if err := auth.CheckPassword(plain, hash); err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
}

func TestCheckPassword_Mismatch(t *testing.T) {
	hash, err := auth.HashPassword("correctPassword123")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if err := auth.CheckPassword("wrongPassword123", hash); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}
