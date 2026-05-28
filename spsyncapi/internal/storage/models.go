package storage

import (
	"time"
)

// Member represents a registered user account.
type Member struct {
	ID           string    `gorm:"primaryKey;type:text"`
	Email        string    `gorm:"uniqueIndex;not null;type:text"`
	PasswordHash string    `gorm:"not null;type:text"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
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

// BucketType identifies the object storage backend.
const (
	BucketTypeS3    = "s3"
	BucketTypeAzure = "azure"
)

// BucketStore holds object storage connection details for backup targets.
// ConfigEncrypted holds AES-GCM ciphertext of JSON config; plaintext is never persisted.
type BucketStore struct {
	ID              string    `gorm:"primaryKey;type:text"`
	MemberID        string    `gorm:"column:member_id;not null;index;type:text;uniqueIndex:idx_bucket_member_name"`
	BucketName      string    `gorm:"column:bucket_name;not null;type:text;uniqueIndex:idx_bucket_member_name"`
	BucketType      string    `gorm:"column:bucket_type;not null;type:text"`
	ConfigEncrypted string    `gorm:"column:config_encrypted;not null;type:text"`
	Active          bool      `gorm:"not null;default:true;index"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

// Organization stores Microsoft Entra / SharePoint tenant connection details.
// TenantSecretEncrypted holds AES-GCM ciphertext; the plaintext is never persisted.
type Organization struct {
	ID                    string    `gorm:"primaryKey;type:text"`
	MemberID              string    `gorm:"column:member_id;not null;index;type:text;uniqueIndex:idx_org_member_tenant"`
	Name                  string    `gorm:"not null;type:text"`
	TenantID              string    `gorm:"column:tenant_id;not null;type:text;uniqueIndex:idx_org_member_tenant"`
	ClientID              string    `gorm:"column:client_id;not null;type:text"`
	TenantSecretEncrypted string    `gorm:"column:tenant_secret_encrypted;not null;type:text"`
	Active                bool      `gorm:"not null;default:true;index"`
	CreatedAt             time.Time `gorm:"not null"`
	UpdatedAt             time.Time `gorm:"not null"`
}

// BackupJob defines a scheduled backup configuration and execution window metadata.
type BackupJob struct {
	ID       string `gorm:"primaryKey;type:text"`
	MemberID string `gorm:"column:member_id;not null;index;type:text"`

	LastRun *time.Time `gorm:"column:last_run;type:datetime"`
	NextRun *time.Time `gorm:"column:next_run;type:datetime"`
	StartAt *time.Time `gorm:"column:start_at;type:datetime"`
	EndAt   *time.Time `gorm:"column:end_at;type:datetime"`

	ScheduleIntervalSeconds *int64     `gorm:"column:schedule_interval_seconds"`
	ScheduleCron            *string    `gorm:"column:schedule_cron;type:text"`
	ScheduleOneTime         *time.Time `gorm:"column:schedule_one_time;type:datetime"`

	Active bool `gorm:"not null;default:true;index"`

	OrganizationID string `gorm:"column:organization_id;not null;type:text;index"`
	BucketStoreID  string `gorm:"column:bucket_store_id;not null;type:text;index"`
	SharePointSite string `gorm:"column:share_point_site;not null;type:text"`

	FilterDocumentLibraries string     `gorm:"column:filter_document_libraries;type:text"`
	FilterMinFileSize       *int64     `gorm:"column:filter_min_file_size"`
	FilterMaxFileSize       *int64     `gorm:"column:filter_max_file_size"`
	FilterCreatedAfter      *time.Time `gorm:"column:filter_created_after;type:datetime"`
	FilterUpdatedAfter      *time.Time `gorm:"column:filter_updated_after;type:datetime"`
	FilterCreatedBefore     *time.Time `gorm:"column:filter_created_before;type:datetime"`
	FilterUpdatedBefore     *time.Time `gorm:"column:filter_updated_before;type:datetime"`

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// BackupRun records a single execution of a backup job.
type BackupRun struct {
	ID        string     `gorm:"primaryKey;type:text"`
	JobID     string     `gorm:"column:job_id;not null;index;type:text"`
	MemberID  string     `gorm:"column:member_id;not null;index;type:text"`
	StartAt   *time.Time `gorm:"column:start_at;type:datetime"`
	EndAt     *time.Time `gorm:"column:end_at;type:datetime"`
	CreatedAt time.Time  `gorm:"not null;index"`
}

// BackupRunFileTransfer records one file transferred during a backup run.
type BackupRunFileTransfer struct {
	ID        string     `gorm:"primaryKey;type:text"`
	RunID     string     `gorm:"column:run_id;not null;index;type:text"`
	FilePath  string     `gorm:"column:file_path;not null;type:text"`
	StartAt   *time.Time `gorm:"column:start_at;type:datetime"`
	EndAt     *time.Time `gorm:"column:end_at;type:datetime"`
	CreatedAt time.Time  `gorm:"not null"`
}

// RestoreJob defines a restore configuration (minimal model for run ownership).
type RestoreJob struct {
	ID       string `gorm:"primaryKey;type:text"`
	MemberID string `gorm:"column:member_id;not null;index;type:text"`

	StartAt  *time.Time `gorm:"column:start_at;type:datetime"`
	LastRun  *time.Time `gorm:"column:last_run;type:datetime"`
	Active   bool       `gorm:"not null;default:true;index"`

	OrganizationID string `gorm:"column:organization_id;not null;type:text;index"`
	BucketStoreID  string `gorm:"column:bucket_store_id;not null;type:text;index"`
	SharePointSite string `gorm:"column:share_point_site;not null;type:text"`

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// RestoreRun records a single execution of a restore job.
type RestoreRun struct {
	ID        string     `gorm:"primaryKey;type:text"`
	JobID     string     `gorm:"column:job_id;not null;index;type:text"`
	MemberID  string     `gorm:"column:member_id;not null;index;type:text"`
	StartAt   *time.Time `gorm:"column:start_at;type:datetime"`
	EndAt     *time.Time `gorm:"column:end_at;type:datetime"`
	CreatedAt time.Time  `gorm:"not null;index"`
}

// RestoreRunFileTransfer records one file transferred during a restore run.
type RestoreRunFileTransfer struct {
	ID        string     `gorm:"primaryKey;type:text"`
	RunID     string     `gorm:"column:run_id;not null;index;type:text"`
	FilePath  string     `gorm:"column:file_path;not null;type:text"`
	StartAt   *time.Time `gorm:"column:start_at;type:datetime"`
	EndAt     *time.Time `gorm:"column:end_at;type:datetime"`
	CreatedAt time.Time  `gorm:"not null"`
}
