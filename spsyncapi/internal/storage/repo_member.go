package storage

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ErrMemberNotFound is returned when a member lookup yields no result.
var ErrMemberNotFound = errors.New("member not found")

// ErrEmailTaken is returned when a registration attempts to use an already-registered email.
var ErrEmailTaken = errors.New("email already registered")

// MemberRepository provides persistence operations for Member records.
type MemberRepository struct {
	db *gorm.DB
}

// NewMemberRepository constructs a MemberRepository backed by db.
func NewMemberRepository(db *gorm.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create inserts a new Member. Email is normalised to lower-case before insert.
// Returns ErrEmailTaken when the email is already registered.
func (r *MemberRepository) Create(m *Member) error {
	m.Email = normaliseEmail(m.Email)
	if err := r.db.Create(m).Error; err != nil {
		if isUniqueConstraintErr(err) {
			return ErrEmailTaken
		}
		return fmt.Errorf("member repo: create: %w", err)
	}
	return nil
}

// FindByEmail returns the member whose email matches (case-insensitive).
// Returns ErrMemberNotFound when no record exists.
func (r *MemberRepository) FindByEmail(email string) (*Member, error) {
	var m Member
	err := r.db.Where("email = ?", normaliseEmail(email)).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("member repo: find by email: %w", err)
	}
	return &m, nil
}

// FindByID returns the member with the given ID.
// Returns ErrMemberNotFound when no record exists.
func (r *MemberRepository) FindByID(id string) (*Member, error) {
	var m Member
	err := r.db.First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("member repo: find by id: %w", err)
	}
	return &m, nil
}

// UpdatePassword sets a new password hash for the given member ID.
func (r *MemberRepository) UpdatePassword(memberID, newHash string) error {
	result := r.db.Model(&Member{}).
		Where("id = ?", memberID).
		Update("password_hash", newHash)
	if result.Error != nil {
		return fmt.Errorf("member repo: update password: %w", result.Error)
	}
	return nil
}

// normaliseEmail trims whitespace and converts to lower-case.
func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isUniqueConstraintErr returns true for SQLite UNIQUE constraint violation errors.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "unique constraint")
}
