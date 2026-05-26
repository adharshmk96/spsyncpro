package crypto_test

import (
	"testing"

	"spsyncapi/internal/crypto"
)

func TestSecretEncryptor_RoundTrip(t *testing.T) {
	enc, err := crypto.NewSecretEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}

	plain := "super-secret-tenant-value"
	ciphertext, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ciphertext == plain {
		t.Fatal("ciphertext must not equal plaintext")
	}

	got, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("got %q, want %q", got, plain)
	}
}

func TestSecretEncryptor_EmptySecret(t *testing.T) {
	_, err := crypto.NewSecretEncryptor("")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}
