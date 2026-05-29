package storage

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrRestoreRunNotFound is returned when a restore run lookup yields no result.
var ErrRestoreRunNotFound = errors.New("restore run not found")

// RestoreRunRepository provides persistence operations for restore runs.
type RestoreRunRepository struct {
	db *gorm.DB
}

// NewRestoreRunRepository constructs a RestoreRunRepository backed by db.
func NewRestoreRunRepository(db *gorm.DB) *RestoreRunRepository {
	return &RestoreRunRepository{db: db}
}

// Update persists changes to a restore run row.
func (r *RestoreRunRepository) Update(run *RestoreRun) error {
	if err := r.db.Save(run).Error; err != nil {
		return fmt.Errorf("restore run repo: update: %w", err)
	}
	return nil
}

// FindIncompleteByJobID returns an in-progress run for jobID, or nil if none.
func (r *RestoreRunRepository) FindIncompleteByJobID(jobID string) (*RestoreRun, error) {
	var run RestoreRun
	err := r.db.Where("job_id = ? AND end_at IS NULL", jobID).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("restore run repo: find incomplete by job: %w", err)
	}
	return &run, nil
}

// ListIncomplete returns restore runs that have not finished (end_at IS NULL).
func (r *RestoreRunRepository) ListIncomplete() ([]RestoreRun, error) {
	var runs []RestoreRun
	err := r.db.Where("end_at IS NULL").Find(&runs).Error
	if err != nil {
		return nil, fmt.Errorf("restore run repo: list incomplete: %w", err)
	}
	return runs, nil
}

// FindFileTransferByRunAndPath returns a file transfer for idempotent activity retries.
func (r *RestoreRunRepository) FindFileTransferByRunAndPath(runID, filePath string) (*RestoreRunFileTransfer, error) {
	var ft RestoreRunFileTransfer
	err := r.db.Where("run_id = ? AND file_path = ?", runID, filePath).First(&ft).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("restore run repo: find file transfer: %w", err)
	}
	return &ft, nil
}

// Create inserts a new restore run.
func (r *RestoreRunRepository) Create(run *RestoreRun) error {
	if err := r.db.Create(run).Error; err != nil {
		return fmt.Errorf("restore run repo: create: %w", err)
	}
	return nil
}

// CreateFileTransfer inserts a file transfer row for a restore run.
func (r *RestoreRunRepository) CreateFileTransfer(ft *RestoreRunFileTransfer) error {
	if err := r.db.Create(ft).Error; err != nil {
		return fmt.Errorf("restore run repo: create file transfer: %w", err)
	}
	return nil
}

// FindByID returns a restore run owned by memberID.
func (r *RestoreRunRepository) FindByID(id, memberID string) (*RestoreRun, error) {
	var run RestoreRun
	err := r.db.Where("id = ? AND member_id = ?", id, memberID).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRestoreRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("restore run repo: find by id: %w", err)
	}
	return &run, nil
}

// List returns restore runs for memberID ordered by created_at descending.
func (r *RestoreRunRepository) List(memberID string, jobID *string, offset, limit int) ([]RestoreRun, int64, error) {
	q := r.db.Model(&RestoreRun{}).Where("member_id = ?", memberID)
	if jobID != nil && *jobID != "" {
		q = q.Where("job_id = ?", *jobID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("restore run repo: count: %w", err)
	}

	var runs []RestoreRun
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("restore run repo: list: %w", err)
	}
	return runs, total, nil
}

// ListFileTransfers returns paginated file transfers for a run ordered by file_path.
func (r *RestoreRunRepository) ListFileTransfers(runID string, offset, limit int) ([]RestoreRunFileTransfer, int64, error) {
	q := r.db.Model(&RestoreRunFileTransfer{}).Where("run_id = ?", runID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("restore run repo: count file transfers: %w", err)
	}

	var transfers []RestoreRunFileTransfer
	err := q.Order("file_path ASC").Offset(offset).Limit(limit).Find(&transfers).Error
	if err != nil {
		return nil, 0, fmt.Errorf("restore run repo: list file transfers: %w", err)
	}
	return transfers, total, nil
}
