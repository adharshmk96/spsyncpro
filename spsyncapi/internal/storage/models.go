package storage

import (
	"time"
)

// Member represents a registered user account.
type Member struct {
	ID           string     `gorm:"primaryKey;type:text"`
	Email        string     `gorm:"uniqueIndex;not null;type:text"`
	PasswordHash string     `gorm:"not null;type:text"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

// Session represents an active login session for a member.
// RevokedAt being non-nil means the session has been explicitly invalidated.
type Session struct {
	ID        string     `gorm:"primaryKey;type:text"`
	MemberID  string     `gorm:"not null;index;type:text"`
	CreatedAt time.Time  `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time `gorm:"type:datetime"`
}

// IsActive returns true when the session has not been revoked and has not expired.
func (s *Session) IsActive() bool {
	return s.RevokedAt == nil && time.Now().Before(s.ExpiresAt)
}

// PasswordResetToken represents a one-time token for resetting a member's password.
// Only the hash is stored; the raw token is sent to the user and never persisted.
type PasswordResetToken struct {
	ID        string     `gorm:"primaryKey;type:text"`
	MemberID  string     `gorm:"not null;index;type:text"`
	TokenHash string     `gorm:"not null;uniqueIndex;type:text"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"type:datetime"`
	CreatedAt time.Time  `gorm:"not null"`
}

// IsValid returns true when the token has not been used and has not expired.
func (p *PasswordResetToken) IsValid() bool {
	return p.UsedAt == nil && time.Now().Before(p.ExpiresAt)
}

// Organization stores Microsoft Entra / SharePoint tenant connection details.
// TenantSecretEncrypted holds AES-GCM ciphertext; the plaintext is never persisted.
type Organization struct {
	ID                    string    `gorm:"primaryKey;type:text"`
	Name                  string    `gorm:"not null;type:text"`
	TenantID              string    `gorm:"column:tenant_id;not null;uniqueIndex;type:text"`
	ClientID              string    `gorm:"column:client_id;not null;type:text"`
	TenantSecretEncrypted string    `gorm:"column:tenant_secret_encrypted;not null;type:text"`
	Active                bool      `gorm:"not null;default:true;index"`
	CreatedAt             time.Time `gorm:"not null"`
	UpdatedAt             time.Time `gorm:"not null"`
}
