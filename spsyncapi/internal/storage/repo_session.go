package storage

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrSessionNotFound is returned when a session lookup yields no result.
var ErrSessionNotFound = errors.New("session not found")

// SessionRepository provides persistence operations for Session records.
type SessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository constructs a SessionRepository backed by db.
func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create inserts a new Session record.
func (r *SessionRepository) Create(s *Session) error {
	if err := r.db.Create(s).Error; err != nil {
		return fmt.Errorf("session repo: create: %w", err)
	}
	return nil
}

// FindByID returns the session with the given ID.
// Returns ErrSessionNotFound when no record exists.
func (r *SessionRepository) FindByID(id string) (*Session, error) {
	var s Session
	err := r.db.First(&s, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("session repo: find by id: %w", err)
	}
	return &s, nil
}

// Revoke marks the session as revoked by setting RevokedAt to now.
func (r *SessionRepository) Revoke(sessionID string) error {
	now := time.Now()
	result := r.db.Model(&Session{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("session repo: revoke: %w", result.Error)
	}
	return nil
}

// RevokeAllForMember marks every active session for the given member as revoked.
func (r *SessionRepository) RevokeAllForMember(memberID string) error {
	now := time.Now()
	result := r.db.Model(&Session{}).
		Where("member_id = ? AND revoked_at IS NULL", memberID).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("session repo: revoke all for member: %w", result.Error)
	}
	return nil
}

// RevokeOtherSessions marks every active session for the given member as revoked
// except the session identified by keepSessionID.
func (r *SessionRepository) RevokeOtherSessions(memberID, keepSessionID string) error {
	now := time.Now()
	result := r.db.Model(&Session{}).
		Where("member_id = ? AND id != ? AND revoked_at IS NULL", memberID, keepSessionID).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("session repo: revoke other sessions: %w", result.Error)
	}
	return nil
}
