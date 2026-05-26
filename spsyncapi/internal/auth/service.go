package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

// --- domain errors ---------------------------------------------------------

var (
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrInvalidPassword  = errors.New("invalid credentials")
	ErrAccountNotFound  = errors.New("account not found")
	ErrSessionInactive  = errors.New("session is inactive")
)

// --- request/response types -----------------------------------------------

// RegisterInput holds the data required to create a new account.
type RegisterInput struct {
	Email    string
	Password string
}

// LoginInput holds the data required to start a session.
type LoginInput struct {
	Email    string
	Password string
}

// TokenResult is returned after a successful register or login.
type TokenResult struct {
	Token string
}

// MemberDetails is returned by the Me endpoint.
type MemberDetails struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ForgotPasswordInput holds the email address to send a reset link to.
type ForgotPasswordInput struct {
	Email string
}

// ResetPasswordInput holds the fields required to complete a password reset.
type ResetPasswordInput struct {
	Email    string
	Token    string
	Password string
}

// ChangePasswordInput holds the fields required to change a password.
type ChangePasswordInput struct {
	MemberID        string
	SessionID       string
	CurrentPassword string
	NewPassword     string
}

// --- service ---------------------------------------------------------------

// Service orchestrates all authentication business logic.
type Service struct {
	members       *storage.MemberRepository
	sessions      *storage.SessionRepository
	resets        *storage.PasswordResetRepository
	jwtCfg        JWTConfig
	sessionTTL    time.Duration
	resetTTL      time.Duration
	frontendBase  string // base URL used in logged reset links
	logger        *slog.Logger
}

// ServiceConfig groups the dependencies needed to build a Service.
type ServiceConfig struct {
	Members      *storage.MemberRepository
	Sessions     *storage.SessionRepository
	Resets       *storage.PasswordResetRepository
	JWTConfig    JWTConfig
	SessionTTL   time.Duration
	ResetTTL     time.Duration
	FrontendBase string // e.g. "https://app.example.com"
	Logger       *slog.Logger
}

// NewService constructs and validates an auth Service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Members == nil {
		return nil, errors.New("auth service: members repository is required")
	}
	if cfg.Sessions == nil {
		return nil, errors.New("auth service: sessions repository is required")
	}
	if cfg.Resets == nil {
		return nil, errors.New("auth service: resets repository is required")
	}
	if len(cfg.JWTConfig.Secret) == 0 {
		return nil, errors.New("auth service: JWT secret is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("auth service: logger is required")
	}
	base := cfg.FrontendBase
	if base == "" {
		base = "http://localhost:3000"
	}
	return &Service{
		members:      cfg.Members,
		sessions:     cfg.Sessions,
		resets:       cfg.Resets,
		jwtCfg:       cfg.JWTConfig,
		sessionTTL:   cfg.SessionTTL,
		resetTTL:     cfg.ResetTTL,
		frontendBase: base,
		logger:       cfg.Logger,
	}, nil
}

// Register creates a new member account and returns an access token.
func (s *Service) Register(in RegisterInput) (*TokenResult, error) {
	if err := validateEmail(in.Email); err != nil {
		return nil, err
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	member := &storage.Member{
		ID:           uuid.NewString(),
		Email:        strings.ToLower(strings.TrimSpace(in.Email)),
		PasswordHash: hash,
	}

	if err := s.members.Create(member); err != nil {
		if errors.Is(err, storage.ErrEmailTaken) {
			return nil, storage.ErrEmailTaken
		}
		return nil, fmt.Errorf("register: %w", err)
	}

	s.logger.Info("member registered", "member_id", member.ID)

	return s.createSession(member.ID)
}

// Login verifies credentials and returns an access token.
func (s *Service) Login(in LoginInput) (*TokenResult, error) {
	if err := validateEmail(in.Email); err != nil {
		return nil, err
	}

	member, err := s.members.FindByEmail(in.Email)
	if errors.Is(err, storage.ErrMemberNotFound) {
		return nil, ErrInvalidPassword
	}
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	if err := CheckPassword(in.Password, member.PasswordHash); err != nil {
		s.logger.Warn("failed login attempt", "member_id", member.ID)
		return nil, ErrInvalidPassword
	}

	s.logger.Info("member logged in", "member_id", member.ID)

	return s.createSession(member.ID)
}

// Me returns the public details of the authenticated member.
func (s *Service) Me(memberID string) (*MemberDetails, error) {
	member, err := s.members.FindByID(memberID)
	if errors.Is(err, storage.ErrMemberNotFound) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("me: %w", err)
	}
	return &MemberDetails{
		ID:        member.ID,
		Email:     member.Email,
		CreatedAt: member.CreatedAt,
	}, nil
}

// Logout revokes the current session.
func (s *Service) Logout(sessionID string) error {
	if err := s.sessions.Revoke(sessionID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	s.logger.Info("session revoked", "session_id", sessionID)
	return nil
}

// ForgotPassword generates a password-reset token for the given email.
// To prevent account enumeration the function always returns success;
// if the email does not exist it simply logs at debug.
func (s *Service) ForgotPassword(in ForgotPasswordInput) error {
	if err := validateEmail(in.Email); err != nil {
		return err
	}

	member, err := s.members.FindByEmail(in.Email)
	if errors.Is(err, storage.ErrMemberNotFound) {
		s.logger.Debug("forgot-password: email not found (no-op)", "email", in.Email)
		return nil
	}
	if err != nil {
		return fmt.Errorf("forgot password: %w", err)
	}

	rawToken, tokenHash, err := generateResetToken()
	if err != nil {
		return fmt.Errorf("forgot password: generate token: %w", err)
	}

	record := &storage.PasswordResetToken{
		ID:        uuid.NewString(),
		MemberID:  member.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.resetTTL),
	}
	if err := s.resets.Create(record); err != nil {
		return fmt.Errorf("forgot password: store token: %w", err)
	}

	// Log the reset link (dev placeholder for real email delivery).
	resetURL := fmt.Sprintf("%s/reset-password?token=%s&email=%s",
		s.frontendBase, rawToken, member.Email)
	s.logger.Info("password reset link generated (dev: send this by email)",
		"member_id", member.ID,
		"reset_url", resetURL,
		"expires_at", record.ExpiresAt,
	)

	return nil
}

// ResetPassword verifies the reset token, sets a new password, and revokes all sessions.
func (s *Service) ResetPassword(in ResetPasswordInput) error {
	if err := validateEmail(in.Email); err != nil {
		return err
	}
	if strings.TrimSpace(in.Token) == "" {
		return errors.New("reset token is required")
	}

	tokenHash := hashResetToken(in.Token)

	record, err := s.resets.FindByTokenHash(tokenHash)
	if errors.Is(err, storage.ErrResetTokenNotFound) {
		return errors.New("invalid or expired reset token")
	}
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}

	// Verify the token belongs to the supplied email (defense in depth).
	member, err := s.members.FindByID(record.MemberID)
	if err != nil {
		return fmt.Errorf("reset password: look up member: %w", err)
	}
	if !strings.EqualFold(member.Email, strings.TrimSpace(in.Email)) {
		return errors.New("invalid or expired reset token")
	}

	newHash, err := HashPassword(in.Password)
	if err != nil {
		return err
	}

	if err := s.members.UpdatePassword(member.ID, newHash); err != nil {
		return fmt.Errorf("reset password: update hash: %w", err)
	}
	if err := s.resets.MarkUsed(record.ID); err != nil {
		return fmt.Errorf("reset password: mark token used: %w", err)
	}
	if err := s.sessions.RevokeAllForMember(member.ID); err != nil {
		return fmt.Errorf("reset password: revoke sessions: %w", err)
	}

	s.logger.Info("password reset completed", "member_id", member.ID)

	return nil
}

// ChangePassword verifies the current password and sets a new one.
// All sessions other than the current one are revoked.
func (s *Service) ChangePassword(in ChangePasswordInput) error {
	member, err := s.members.FindByID(in.MemberID)
	if errors.Is(err, storage.ErrMemberNotFound) {
		return ErrAccountNotFound
	}
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	if err := CheckPassword(in.CurrentPassword, member.PasswordHash); err != nil {
		return ErrInvalidPassword
	}

	newHash, err := HashPassword(in.NewPassword)
	if err != nil {
		return err
	}

	if err := s.members.UpdatePassword(member.ID, newHash); err != nil {
		return fmt.Errorf("change password: update hash: %w", err)
	}

	// Revoke every session except the currently active one so the caller
	// stays logged in after changing their password.
	if err := s.revokeOtherSessions(member.ID, in.SessionID); err != nil {
		return fmt.Errorf("change password: revoke other sessions: %w", err)
	}

	s.logger.Info("password changed", "member_id", member.ID)

	return nil
}

// RefreshForSession mints a new access token for an already-authenticated session.
// It is called by the auth middleware when the JWT is expired but the session is valid.
func (s *Service) RefreshForSession(sessionID, memberID string) (string, error) {
	token, err := MintToken(s.jwtCfg, memberID, sessionID)
	if err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}
	return token, nil
}

// LookupSession returns the session row for the given ID, or an error if the
// session cannot be found or is inactive.
func (s *Service) LookupSession(sessionID string) (*storage.Session, error) {
	sess, err := s.sessions.FindByID(sessionID)
	if errors.Is(err, storage.ErrSessionNotFound) {
		return nil, ErrSessionInactive
	}
	if err != nil {
		return nil, err
	}
	if !sess.IsActive() {
		return nil, ErrSessionInactive
	}
	return sess, nil
}

// --- helpers ---------------------------------------------------------------

// createSession inserts a new session record and mints a JWT.
func (s *Service) createSession(memberID string) (*TokenResult, error) {
	sess := &storage.Session{
		ID:        uuid.NewString(),
		MemberID:  memberID,
		ExpiresAt: time.Now().Add(s.sessionTTL),
	}
	if err := s.sessions.Create(sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	token, err := MintToken(s.jwtCfg, memberID, sess.ID)
	if err != nil {
		return nil, err
	}
	return &TokenResult{Token: token}, nil
}

// revokeOtherSessions revokes all sessions for memberID except keepSessionID.
func (s *Service) revokeOtherSessions(memberID, keepSessionID string) error {
	// Revoke all, then we don't need a special "except" query; session lookup
	// will fail for the revoked ones.  However the caller's in-flight request
	// already has the memberId in context so it will succeed.
	// Trade-off: caller must re-authenticate on next request after change-password.
	// For a better UX, revoke all except keepSessionID.
	// Here we implement the targeted approach via raw GORM update.
	return s.sessions.RevokeOtherSessions(memberID, keepSessionID)
}

// validateEmail returns an error when the email is not a valid RFC 5322 address.
func validateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

// generateResetToken creates a cryptographically random 32-byte token, returning
// the hex-encoded raw token (to send to the user) and its SHA-256 hex hash (to store).
func generateResetToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	hash = hashResetToken(raw)
	return raw, hash, nil
}

// hashResetToken returns the hex-encoded SHA-256 of a raw reset token.
func hashResetToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
