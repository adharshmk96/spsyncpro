package storage

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrRestoreJobNotFound is returned when a restore job lookup yields no active result.
var ErrRestoreJobNotFound = errors.New("restore job not found")

// RestoreJobRepository provides persistence operations for RestoreJob records.
type RestoreJobRepository struct {
	db *gorm.DB
}

// NewRestoreJobRepository constructs a RestoreJobRepository backed by db.
func NewRestoreJobRepository(db *gorm.DB) *RestoreJobRepository {
	return &RestoreJobRepository{db: db}
}

// Create inserts a new restore job.
func (r *RestoreJobRepository) Create(job *RestoreJob) error {
	if err := r.db.Create(job).Error; err != nil {
		return fmt.Errorf("restore job repo: create: %w", err)
	}
	return nil
}

// FindActiveByID returns an active restore job owned by memberID.
func (r *RestoreJobRepository) FindActiveByID(id, memberID string) (*RestoreJob, error) {
	var job RestoreJob
	err := r.db.Where("id = ? AND active = ? AND member_id = ?", id, true, memberID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRestoreJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("restore job repo: find by id: %w", err)
	}
	return &job, nil
}

// ListActive returns active restore jobs owned by memberID ordered by created date descending.
func (r *RestoreJobRepository) ListActive(memberID string) ([]RestoreJob, error) {
	var jobs []RestoreJob
	err := r.db.Where("active = ? AND member_id = ?", true, memberID).Order("created_at DESC").Find(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("restore job repo: list: %w", err)
	}
	return jobs, nil
}

// Update persists changes to an existing restore job row.
func (r *RestoreJobRepository) Update(job *RestoreJob) error {
	result := r.db.Save(job)
	if result.Error != nil {
		return fmt.Errorf("restore job repo: update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRestoreJobNotFound
	}
	return nil
}

// MarkInactive sets active=false for the restore job owned by memberID (soft delete).
func (r *RestoreJobRepository) MarkInactive(id, memberID string) error {
	result := r.db.Model(&RestoreJob{}).
		Where("id = ? AND active = ? AND member_id = ?", id, true, memberID).
		Updates(map[string]interface{}{
			"active":     false,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("restore job repo: mark inactive: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRestoreJobNotFound
	}
	return nil
}
