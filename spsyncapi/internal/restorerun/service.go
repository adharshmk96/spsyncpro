package restorerun

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"spsyncapi/internal/storage"
	"spsyncapi/internal/temporal"

	"github.com/google/uuid"
)

var (
	ErrRestoreRunNotFound = errors.New("restore run not found")
	ErrRestoreJobNotFound = errors.New("restore job not found")
	ErrInvalidMemberID    = errors.New("member id is required")
	ErrRunNotInProgress   = errors.New("restore run is not in progress")
)

// RunExecutor starts and stops Temporal workflows for restore runs.
type RunExecutor interface {
	StartRestoreRun(ctx context.Context, in temporal.RunWorkflowInput) error
	StopRestoreRun(ctx context.Context, runID string) error
}

// RunDetails is the API representation of a restore run.
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

// ListResult holds a paginated list of restore runs.
type ListResult struct {
	Runs  []RunDetails
	Total int64
}

// GetResult holds a restore run and paginated file transfers.
type GetResult struct {
	Run           RunDetails
	FileTransfers []FileTransferDetails
	FilesTotal    int64
}

type Service struct {
	runRepo  *storage.RestoreRunRepository
	jobRepo  *storage.RestoreJobRepository
	executor RunExecutor
	logger   *slog.Logger
}

type ServiceConfig struct {
	RunRepo  *storage.RestoreRunRepository
	JobRepo  *storage.RestoreJobRepository
	Executor RunExecutor
	Logger   *slog.Logger
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.RunRepo == nil {
		return nil, errors.New("restore run repo is required")
	}
	if cfg.JobRepo == nil {
		return nil, errors.New("restore job repo is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	return &Service{
		runRepo:  cfg.RunRepo,
		jobRepo:  cfg.JobRepo,
		executor: cfg.Executor,
		logger:   cfg.Logger,
	}, nil
}

// StartRun creates a restore run and starts its Temporal workflow.
func (s *Service) StartRun(ctx context.Context, memberID, jobID string) (*RunDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if _, err := s.jobRepo.FindActiveByID(jobID, memberID); err != nil {
		if errors.Is(err, storage.ErrRestoreJobNotFound) {
			return nil, ErrRestoreJobNotFound
		}
		return nil, fmt.Errorf("restore run service: validate job: %w", err)
	}

	now := time.Now().UTC()
	run := &storage.RestoreRun{
		ID:        uuid.NewString(),
		JobID:     jobID,
		MemberID:  memberID,
		CreatedAt: now,
	}
	if err := s.runRepo.Create(run); err != nil {
		return nil, fmt.Errorf("restore run service: create run: %w", err)
	}

	if s.executor == nil {
		return nil, fmt.Errorf("restore run service: run executor not configured")
	}
	if err := s.executor.StartRestoreRun(ctx, temporal.RunWorkflowInput{
		RunID:    run.ID,
		JobID:    jobID,
		MemberID: memberID,
		Resume:   true,
	}); err != nil {
		return nil, fmt.Errorf("restore run service: start workflow: %w", err)
	}

	details := toRunDetails(run)
	return &details, nil
}

// StopRun cancels an in-progress restore run workflow.
func (s *Service) StopRun(ctx context.Context, memberID, runID string) (*RunDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	run, err := s.getRunWithJobCheck(memberID, runID)
	if err != nil {
		return nil, err
	}
	if run.EndAt != nil {
		return nil, ErrRunNotInProgress
	}
	if s.executor == nil {
		return nil, fmt.Errorf("restore run service: run executor not configured")
	}
	if err := s.executor.StopRestoreRun(ctx, run.ID); err != nil {
		return nil, fmt.Errorf("restore run service: stop workflow: %w", err)
	}
	details := toRunDetails(run)
	return &details, nil
}

func (s *Service) List(memberID string, jobID *string, page, limit int) (*ListResult, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if jobID != nil && strings.TrimSpace(*jobID) != "" {
		if _, err := s.jobRepo.FindActiveByID(*jobID, memberID); err != nil {
			if errors.Is(err, storage.ErrRestoreJobNotFound) {
				return nil, ErrRestoreJobNotFound
			}
			return nil, fmt.Errorf("restore run service: validate job: %w", err)
		}
	} else {
		jobID = nil
	}

	offset := (page - 1) * limit
	runs, total, err := s.runRepo.List(memberID, jobID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("restore run service: list: %w", err)
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
		return nil, fmt.Errorf("restore run service: list file transfers: %w", err)
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

func (s *Service) getRunWithJobCheck(memberID, runID string) (*storage.RestoreRun, error) {
	run, err := s.runRepo.FindByID(runID, memberID)
	if err != nil {
		if errors.Is(err, storage.ErrRestoreRunNotFound) {
			return nil, ErrRestoreRunNotFound
		}
		return nil, fmt.Errorf("restore run service: find run: %w", err)
	}

	if _, err := s.jobRepo.FindActiveByID(run.JobID, memberID); err != nil {
		if errors.Is(err, storage.ErrRestoreJobNotFound) {
			return nil, ErrRestoreRunNotFound
		}
		return nil, fmt.Errorf("restore run service: validate job: %w", err)
	}

	return run, nil
}

func toRunDetails(run *storage.RestoreRun) RunDetails {
	return RunDetails{
		ID:      run.ID,
		JobID:   run.JobID,
		StartAt: run.StartAt,
		EndAt:   run.EndAt,
	}
}

func toFileTransferDetails(ft *storage.RestoreRunFileTransfer) FileTransferDetails {
	return FileTransferDetails{
		FilePath: ft.FilePath,
		StartAt:  ft.StartAt,
		EndAt:    ft.EndAt,
	}
}
