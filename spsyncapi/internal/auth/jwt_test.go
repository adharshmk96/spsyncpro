package auth_test

import (
	"testing"
	"time"

	"spsyncapi/internal/auth"
)

func testJWTConfig(ttl time.Duration) auth.JWTConfig {
	return auth.JWTConfig{
		Secret:    []byte("test-secret-key"),
		Issuer:    "spsyncapi-test",
		AccessTTL: ttl,
	}
}

func TestMintAndParseToken_Valid(t *testing.T) {
	cfg := testJWTConfig(15 * time.Minute)

	token, err := auth.MintToken(cfg, "member-1", "session-1")
	if err != nil {
		t.Fatalf("mint error: %v", err)
	}

	result, err := auth.ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Expired {
		t.Fatal("expected token to not be expired")
	}
	if result.Claims.Subject != "member-1" {
		t.Errorf("subject: got %q, want %q", result.Claims.Subject, "member-1")
	}
	if result.Claims.SessionID != "session-1" {
		t.Errorf("session id: got %q, want %q", result.Claims.SessionID, "session-1")
	}
}

func TestParseToken_Expired(t *testing.T) {
	// Use a negative TTL to mint a token that is already past its expiry.
	cfg := testJWTConfig(-1 * time.Second)

	token, err := auth.MintToken(cfg, "member-2", "session-2")
	if err != nil {
		t.Fatalf("mint error: %v", err)
	}

	result, err := auth.ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("expected no error for expired (but valid-signature) token, got: %v", err)
	}
	if !result.Expired {
		t.Fatal("expected token to be marked expired")
	}
	if result.Claims.Subject != "member-2" {
		t.Errorf("subject: got %q, want %q", result.Claims.Subject, "member-2")
	}
}

func TestParseToken_InvalidSignature(t *testing.T) {
	cfg := testJWTConfig(15 * time.Minute)

	token, err := auth.MintToken(cfg, "member-3", "session-3")
	if err != nil {
		t.Fatalf("mint error: %v", err)
	}

	// Parse with a different secret to trigger signature failure.
	badCfg := cfg
	badCfg.Secret = []byte("wrong-secret")
	_, err = auth.ParseToken(badCfg, token)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	cfg := testJWTConfig(15 * time.Minute)
	_, err := auth.ParseToken(cfg, "not.a.jwt")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}
