package storage

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrResetTokenNotFound is returned when a password-reset token lookup yields no result.
var ErrResetTokenNotFound = errors.New("reset token not found")

// PasswordResetRepository provides persistence operations for PasswordResetToken records.
type PasswordResetRepository struct {
	db *gorm.DB
}

// NewPasswordResetRepository constructs a PasswordResetRepository backed by db.
func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// Create inserts a new PasswordResetToken record.
func (r *PasswordResetRepository) Create(t *PasswordResetToken) error {
	if err := r.db.Create(t).Error; err != nil {
		return fmt.Errorf("password reset repo: create: %w", err)
	}
	return nil
}

// FindByTokenHash looks up an un-used, un-expired token by its hash.
// Returns ErrResetTokenNotFound when no matching record exists.
func (r *PasswordResetRepository) FindByTokenHash(tokenHash string) (*PasswordResetToken, error) {
	var t PasswordResetToken
	err := r.db.
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResetTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("password reset repo: find by hash: %w", err)
	}
	return &t, nil
}

// MarkUsed sets UsedAt to now for the given token ID.
func (r *PasswordResetRepository) MarkUsed(id string) error {
	now := time.Now()
	result := r.db.Model(&PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", now)
	if result.Error != nil {
		return fmt.Errorf("password reset repo: mark used: %w", result.Error)
	}
	return nil
}
