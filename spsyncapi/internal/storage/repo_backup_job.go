package storage

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrBackupJobNotFound is returned when a backup job lookup yields no active result.
var ErrBackupJobNotFound = errors.New("backup job not found")

// BackupJobRepository provides persistence operations for BackupJob records.
type BackupJobRepository struct {
	db *gorm.DB
}

// NewBackupJobRepository constructs a BackupJobRepository backed by db.
func NewBackupJobRepository(db *gorm.DB) *BackupJobRepository {
	return &BackupJobRepository{db: db}
}

// Create inserts a new BackupJob.
func (r *BackupJobRepository) Create(job *BackupJob) error {
	if err := r.db.Create(job).Error; err != nil {
		return fmt.Errorf("backup job repo: create: %w", err)
	}
	return nil
}

// FindActiveByID returns an active backup job owned by memberID.
func (r *BackupJobRepository) FindActiveByID(id, memberID string) (*BackupJob, error) {
	var job BackupJob
	err := r.db.Where("id = ? AND active = ? AND member_id = ?", id, true, memberID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBackupJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("backup job repo: find by id: %w", err)
	}
	return &job, nil
}

// ListActive returns active backup jobs owned by memberID ordered by created date descending.
func (r *BackupJobRepository) ListActive(memberID string) ([]BackupJob, error) {
	var jobs []BackupJob
	err := r.db.Where("active = ? AND member_id = ?", true, memberID).Order("created_at DESC").Find(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("backup job repo: list: %w", err)
	}
	return jobs, nil
}

// Update persists changes to an existing backup job row.
func (r *BackupJobRepository) Update(job *BackupJob) error {
	result := r.db.Save(job)
	if result.Error != nil {
		return fmt.Errorf("backup job repo: update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrBackupJobNotFound
	}
	return nil
}

// MarkInactive sets active=false for the backup job owned by memberID (soft delete).
func (r *BackupJobRepository) MarkInactive(id, memberID string) error {
	result := r.db.Model(&BackupJob{}).
		Where("id = ? AND active = ? AND member_id = ?", id, true, memberID).
		Updates(map[string]interface{}{
			"active":     false,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("backup job repo: mark inactive: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrBackupJobNotFound
	}
	return nil
}
