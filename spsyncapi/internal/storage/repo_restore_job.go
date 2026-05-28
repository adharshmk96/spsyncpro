package storage

import (
	"errors"
	"fmt"

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
