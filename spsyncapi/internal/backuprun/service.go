package backuprun

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"spsyncapi/internal/storage"
)

var (
	ErrBackupRunNotFound   = errors.New("backup run not found")
	ErrBackupJobNotFound   = errors.New("backup job not found")
	ErrInvalidMemberID = errors.New("member id is required")
)

// RunDetails is the API representation of a backup run.
type RunDetails struct {
	ID      string     `json:"id"`
	JobID   string     `json:"job_id"`
	StartAt *time.Time `json:"start_at,omitempty"`
	EndAt   *time.Time `json:"end_at,omitempty"`
}

// FileTransferDetails is the API representation of a file transfer within a run.
type FileTransferDetails struct {
	FilePath string     `json:"file_path"`
	StartAt  *time.Time `json:"start_at,omitempty"`
	EndAt    *time.Time `json:"end_at,omitempty"`
}

// ListResult holds a paginated list of backup runs.
type ListResult struct {
	Runs  []RunDetails
	Total int64
}

// GetResult holds a backup run and paginated file transfers.
type GetResult struct {
	Run           RunDetails
	FileTransfers []FileTransferDetails
	FilesTotal    int64
}

type Service struct {
	runRepo *storage.BackupRunRepository
	jobRepo *storage.BackupJobRepository
	logger  *slog.Logger
}

type ServiceConfig struct {
	RunRepo *storage.BackupRunRepository
	JobRepo *storage.BackupJobRepository
	Logger  *slog.Logger
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.RunRepo == nil {
		return nil, errors.New("backup run repo is required")
	}
	if cfg.JobRepo == nil {
		return nil, errors.New("backup job repo is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	return &Service{
		runRepo: cfg.RunRepo,
		jobRepo: cfg.JobRepo,
		logger:  cfg.Logger,
	}, nil
}

func (s *Service) List(memberID string, jobID *string, page, limit int) (*ListResult, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if jobID != nil && strings.TrimSpace(*jobID) != "" {
		if _, err := s.jobRepo.FindActiveByID(*jobID, memberID); err != nil {
			if errors.Is(err, storage.ErrBackupJobNotFound) {
				return nil, ErrBackupJobNotFound
			}
			return nil, fmt.Errorf("backup run service: validate job: %w", err)
		}
	} else {
		jobID = nil
	}

	offset := (page - 1) * limit
	runs, total, err := s.runRepo.List(memberID, jobID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("backup run service: list: %w", err)
	}

	details := make([]RunDetails, 0, len(runs))
	for i := range runs {
		details = append(details, toRunDetails(&runs[i]))
	}
	return &ListResult{Runs: details, Total: total}, nil
}

func (s *Service) Get(memberID, runID string, page, limit int) (*GetResult, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}

	run, err := s.getRunWithJobCheck(memberID, runID)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	transfers, filesTotal, err := s.runRepo.ListFileTransfers(run.ID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("backup run service: list file transfers: %w", err)
	}

	files := make([]FileTransferDetails, 0, len(transfers))
	for i := range transfers {
		files = append(files, toFileTransferDetails(&transfers[i]))
	}

	return &GetResult{
		Run:           toRunDetails(run),
		FileTransfers: files,
		FilesTotal:    filesTotal,
	}, nil
}

func (s *Service) getRunWithJobCheck(memberID, runID string) (*storage.BackupRun, error) {
	run, err := s.runRepo.FindByID(runID, memberID)
	if err != nil {
		if errors.Is(err, storage.ErrBackupRunNotFound) {
			return nil, ErrBackupRunNotFound
		}
		return nil, fmt.Errorf("backup run service: find run: %w", err)
	}

	if _, err := s.jobRepo.FindActiveByID(run.JobID, memberID); err != nil {
		if errors.Is(err, storage.ErrBackupJobNotFound) {
			return nil, ErrBackupRunNotFound
		}
		return nil, fmt.Errorf("backup run service: validate job: %w", err)
	}

	return run, nil
}

func toRunDetails(run *storage.BackupRun) RunDetails {
	return RunDetails{
		ID:      run.ID,
		JobID:   run.JobID,
		StartAt: run.StartAt,
		EndAt:   run.EndAt,
	}
}

func toFileTransferDetails(ft *storage.BackupRunFileTransfer) FileTransferDetails {
	return FileTransferDetails{
		FilePath: ft.FilePath,
		StartAt:  ft.StartAt,
		EndAt:    ft.EndAt,
	}
}
