package auth_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/auth"
	"spsyncapi/internal/storage"
)

// openTestDB opens an in-memory SQLite database suitable for unit tests.
func openTestDB(t *testing.T) *storage.MemberRepository {
	t.Helper()
	db, err := storage.Open("file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return storage.NewMemberRepository(db)
}

// newTestService builds a fully wired auth.Service backed by an in-memory SQLite DB.
func newTestService(t *testing.T) *auth.Service {
	t.Helper()

	db, err := storage.Open("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	members := storage.NewMemberRepository(db)
	sessions := storage.NewSessionRepository(db)
	resets := storage.NewPasswordResetRepository(db)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	svc, err := auth.NewService(auth.ServiceConfig{
		Members:  members,
		Sessions: sessions,
		Resets:   resets,
		JWTConfig: auth.JWTConfig{
			Secret:    []byte("test-secret"),
			Issuer:    "test",
			AccessTTL: 15 * time.Minute,
		},
		SessionTTL: 24 * time.Hour,
		ResetTTL:   30 * time.Minute,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func TestRegister_Success(t *testing.T) {
	svc := newTestService(t)

	result, err := svc.Register(auth.RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)

	input := auth.RegisterInput{Email: "dup@example.com", Password: "password123"}
	if _, err := svc.Register(input); err != nil {
		t.Fatalf("first register error: %v", err)
	}

	_, err := svc.Register(input)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestLogin_Success(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.Register(auth.RegisterInput{Email: "login@example.com", Password: "password123"}); err != nil {
		t.Fatalf("register error: %v", err)
	}

	result, err := svc.Login(auth.LoginInput{Email: "login@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.Register(auth.RegisterInput{Email: "pass@example.com", Password: "password123"}); err != nil {
		t.Fatalf("register error: %v", err)
	}

	_, err := svc.Login(auth.LoginInput{Email: "pass@example.com", Password: "wrongpassword"})
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Login(auth.LoginInput{Email: "nobody@example.com", Password: "password123"})
	if err == nil {
		t.Fatal("expected error for unknown email, got nil")
	}
}

func TestMe_Success(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.Register(auth.RegisterInput{Email: "me@example.com", Password: "password123"}); err != nil {
		t.Fatalf("register error: %v", err)
	}

	// Login to get the token, parse it to get the member ID.
	loginResult, err := svc.Login(auth.LoginInput{Email: "me@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login error: %v", err)
	}

	jwtCfg := auth.JWTConfig{Secret: []byte("test-secret"), Issuer: "test", AccessTTL: 15 * time.Minute}
	parsed, err := auth.ParseToken(jwtCfg, loginResult.Token)
	if err != nil {
		t.Fatalf("parse token error: %v", err)
	}

	details, err := svc.Me(parsed.Claims.Subject)
	if err != nil {
		t.Fatalf("me error: %v", err)
	}
	if details.Email != "me@example.com" {
		t.Errorf("email: got %q, want %q", details.Email, "me@example.com")
	}
}
