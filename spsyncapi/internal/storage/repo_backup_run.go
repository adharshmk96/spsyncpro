package storage

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrBackupRunNotFound is returned when a backup run lookup yields no result.
var ErrBackupRunNotFound = errors.New("backup run not found")

// BackupRunRepository provides persistence operations for backup runs.
type BackupRunRepository struct {
	db *gorm.DB
}

// NewBackupRunRepository constructs a BackupRunRepository backed by db.
func NewBackupRunRepository(db *gorm.DB) *BackupRunRepository {
	return &BackupRunRepository{db: db}
}

// Update persists changes to a backup run row.
func (r *BackupRunRepository) Update(run *BackupRun) error {
	if err := r.db.Save(run).Error; err != nil {
		return fmt.Errorf("backup run repo: update: %w", err)
	}
	return nil
}

// ListIncomplete returns backup runs that have not finished (end_at IS NULL).
func (r *BackupRunRepository) ListIncomplete() ([]BackupRun, error) {
	var runs []BackupRun
	err := r.db.Where("end_at IS NULL").Find(&runs).Error
	if err != nil {
		return nil, fmt.Errorf("backup run repo: list incomplete: %w", err)
	}
	return runs, nil
}

// FindFileTransferByRunAndPath returns a file transfer for idempotent activity retries.
func (r *BackupRunRepository) FindFileTransferByRunAndPath(runID, filePath string) (*BackupRunFileTransfer, error) {
	var ft BackupRunFileTransfer
	err := r.db.Where("run_id = ? AND file_path = ?", runID, filePath).First(&ft).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backup run repo: find file transfer: %w", err)
	}
	return &ft, nil
}

// Create inserts a new backup run.
func (r *BackupRunRepository) Create(run *BackupRun) error {
	if err := r.db.Create(run).Error; err != nil {
		return fmt.Errorf("backup run repo: create: %w", err)
	}
	return nil
}

// CreateFileTransfer inserts a file transfer row for a backup run.
func (r *BackupRunRepository) CreateFileTransfer(ft *BackupRunFileTransfer) error {
	if err := r.db.Create(ft).Error; err != nil {
		return fmt.Errorf("backup run repo: create file transfer: %w", err)
	}
	return nil
}

// FindByID returns a backup run owned by memberID.
func (r *BackupRunRepository) FindByID(id, memberID string) (*BackupRun, error) {
	var run BackupRun
	err := r.db.Where("id = ? AND member_id = ?", id, memberID).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBackupRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("backup run repo: find by id: %w", err)
	}
	return &run, nil
}

// List returns backup runs for memberID ordered by created_at descending.
func (r *BackupRunRepository) List(memberID string, jobID *string, offset, limit int) ([]BackupRun, int64, error) {
	q := r.db.Model(&BackupRun{}).Where("member_id = ?", memberID)
	if jobID != nil && *jobID != "" {
		q = q.Where("job_id = ?", *jobID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("backup run repo: count: %w", err)
	}

	var runs []BackupRun
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("backup run repo: list: %w", err)
	}
	return runs, total, nil
}

// ListFileTransfers returns paginated file transfers for a run ordered by file_path.
func (r *BackupRunRepository) ListFileTransfers(runID string, offset, limit int) ([]BackupRunFileTransfer, int64, error) {
	q := r.db.Model(&BackupRunFileTransfer{}).Where("run_id = ?", runID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("backup run repo: count file transfers: %w", err)
	}

	var transfers []BackupRunFileTransfer
	err := q.Order("file_path ASC").Offset(offset).Limit(limit).Find(&transfers).Error
	if err != nil {
		return nil, 0, fmt.Errorf("backup run repo: list file transfers: %w", err)
	}
	return transfers, total, nil
}
