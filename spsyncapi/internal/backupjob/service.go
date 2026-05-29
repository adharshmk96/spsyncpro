package backupjob

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

var (
	ErrBackupJobNotFound      = errors.New("backup job not found")
	ErrInvalidSchedule        = errors.New("schedule must set exactly one of interval, cron, or one_time")
	ErrInvalidInterval        = errors.New("schedule.interval must be greater than 0")
	ErrInvalidCron            = errors.New("schedule.cron must be a valid cron expression")
	ErrInvalidOneTime         = errors.New("schedule.one_time must be in the future")
	ErrInvalidStartAt         = errors.New("start_at must be before end_at")
	ErrInvalidOrganizationID  = errors.New("job_config.organization is required")
	ErrInvalidBucketStoreID   = errors.New("job_config.bucket_store is required")
	ErrInvalidSharePointSite  = errors.New("job_config.share_point_site is required")
	ErrInvalidMinFileSize     = errors.New("filters.min_file_size must be >= 0")
	ErrInvalidMaxFileSize     = errors.New("filters.max_file_size must be >= 0")
	ErrInvalidFileSizeRange   = errors.New("filters.max_file_size must be >= filters.min_file_size")
	ErrInvalidCreatedRange    = errors.New("filters.created_before must be after created_after")
	ErrInvalidUpdatedRange    = errors.New("filters.updated_before must be after updated_after")
	ErrInvalidDocumentLibrary = errors.New("filters.document_libraries contains empty value")
	ErrInvalidMemberID        = errors.New("member id is required")
)

type ScheduleInput struct {
	Type            string // "one_time" | "recurring" | ""
	IntervalSeconds *int64
	Cron            *string
	OneTime         *time.Time
}

type FilterInput struct {
	DocumentLibrariesCSV string
	MinFileSize          *int64
	MaxFileSize          *int64
	CreatedAfter         *time.Time
	UpdatedAfter         *time.Time
	CreatedBefore        *time.Time
	UpdatedBefore        *time.Time
}

type JobConfigInput struct {
	OrganizationID string
	BucketStoreID  string
	SharePointSite string
	Filters        FilterInput
}

type CreateInput struct {
	MemberID  string
	LastRun   *time.Time
	NextRun   *time.Time
	StartAt   *time.Time
	EndAt     *time.Time
	Active    bool
	Schedule  ScheduleInput
	JobConfig JobConfigInput
}

type UpdateInput struct {
	ID string

	LastRun   *time.Time
	NextRun   *time.Time
	StartAt   *time.Time
	EndAt     *time.Time
	Active    bool
	Schedule  ScheduleInput
	JobConfig JobConfigInput
}

type ScheduleDetails struct {
	IntervalSeconds *int64     `json:"interval,omitempty"`
	Cron            *string    `json:"cron,omitempty"`
	OneTime         *time.Time `json:"one_time,omitempty"`
}

type FilterDetails struct {
	DocumentLibrariesCSV string     `json:"document_libraries"`
	MinFileSize          *int64     `json:"min_file_size,omitempty"`
	MaxFileSize          *int64     `json:"max_file_size,omitempty"`
	CreatedAfter         *time.Time `json:"created_after,omitempty"`
	UpdatedAfter         *time.Time `json:"updated_after,omitempty"`
	CreatedBefore        *time.Time `json:"created_before,omitempty"`
	UpdatedBefore        *time.Time `json:"updated_before,omitempty"`
}

type JobConfigDetails struct {
	OrganizationID string        `json:"organization"`
	BucketStoreID  string        `json:"bucket_store"`
	SharePointSite string        `json:"share_point_site"`
	Filters        FilterDetails `json:"filters"`
}

type BackupJobDetails struct {
	ID        string           `json:"id"`
	LastRun   *time.Time       `json:"last_run,omitempty"`
	NextRun   *time.Time       `json:"next_run,omitempty"`
	StartAt   *time.Time       `json:"start_at,omitempty"`
	EndAt     *time.Time       `json:"end_at,omitempty"`
	Active    bool             `json:"active"`
	Schedule  ScheduleDetails  `json:"schedule"`
	JobConfig JobConfigDetails `json:"job_config"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// ScheduleSyncer syncs backup jobs to Temporal schedules.
type ScheduleSyncer interface {
	SyncJob(ctx context.Context, job *storage.BackupJob) error
	DeleteJobSchedule(ctx context.Context, jobID string) error
}

// RunStarter starts backup runs for a job (immediate or delayed).
type RunStarter interface {
	StartRun(ctx context.Context, memberID, jobID string) error
	StartRunAt(ctx context.Context, memberID, jobID string, at time.Time) error
}

type Service struct {
	repo           *storage.BackupJobRepository
	orgRepo        *storage.OrganizationRepository
	bucketRepo     *storage.BucketStoreRepository
	scheduleSyncer ScheduleSyncer
	runStarter     RunStarter
	logger         *slog.Logger
}

type ServiceConfig struct {
	Repo           *storage.BackupJobRepository
	OrgRepo        *storage.OrganizationRepository
	BucketRepo     *storage.BucketStoreRepository
	ScheduleSyncer ScheduleSyncer
	RunStarter     RunStarter
	Logger         *slog.Logger
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("backup job repo is required")
	}
	if cfg.OrgRepo == nil {
		return nil, errors.New("organization repo is required")
	}
	if cfg.BucketRepo == nil {
		return nil, errors.New("bucket store repo is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	syncer := cfg.ScheduleSyncer
	if syncer == nil {
		syncer = noopScheduleSyncer{}
	}
	starter := cfg.RunStarter
	if starter == nil {
		starter = noopRunStarter{}
	}
	return &Service{
		repo:           cfg.Repo,
		orgRepo:        cfg.OrgRepo,
		bucketRepo:     cfg.BucketRepo,
		scheduleSyncer: syncer,
		runStarter:     starter,
		logger:         cfg.Logger,
	}, nil
}

type noopScheduleSyncer struct{}

func (noopScheduleSyncer) SyncJob(context.Context, *storage.BackupJob) error { return nil }
func (noopScheduleSyncer) DeleteJobSchedule(context.Context, string) error    { return nil }

type noopRunStarter struct{}

func (noopRunStarter) StartRun(context.Context, string, string) error { return nil }
func (noopRunStarter) StartRunAt(context.Context, string, string, time.Time) error {
	return nil
}

func (s *Service) Create(in CreateInput) (*BackupJobDetails, error) {
	if strings.TrimSpace(in.MemberID) == "" {
		return nil, ErrInvalidMemberID
	}
	normalized, err := normalizeAndValidateInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.validateJobReferences(in.MemberID, normalized.JobConfig); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job := &storage.BackupJob{
		ID:                      uuid.NewString(),
		MemberID:                in.MemberID,
		LastRun:                 normalized.LastRun,
		NextRun:                 normalized.NextRun,
		StartAt:                 normalized.StartAt,
		EndAt:                   normalized.EndAt,
		ScheduleIntervalSeconds: normalized.Schedule.IntervalSeconds,
		ScheduleCron:            normalized.Schedule.Cron,
		ScheduleOneTime:         normalized.Schedule.OneTime,
		Active:                  normalized.Active,
		OrganizationID:          normalized.JobConfig.OrganizationID,
		BucketStoreID:           normalized.JobConfig.BucketStoreID,
		SharePointSite:          normalized.JobConfig.SharePointSite,
		FilterDocumentLibraries: normalized.JobConfig.Filters.DocumentLibrariesCSV,
		FilterMinFileSize:       normalized.JobConfig.Filters.MinFileSize,
		FilterMaxFileSize:       normalized.JobConfig.Filters.MaxFileSize,
		FilterCreatedAfter:      normalized.JobConfig.Filters.CreatedAfter,
		FilterUpdatedAfter:      normalized.JobConfig.Filters.UpdatedAfter,
		FilterCreatedBefore:     normalized.JobConfig.Filters.CreatedBefore,
		FilterUpdatedBefore:     normalized.JobConfig.Filters.UpdatedBefore,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := s.repo.Create(job); err != nil {
		return nil, fmt.Errorf("create backup job: %w", err)
	}
	ctx := context.Background()
	switch {
	case isImmediateOneTime(normalized.Schedule):
		if err := s.runStarter.StartRun(ctx, in.MemberID, job.ID); err != nil {
			return nil, fmt.Errorf("start immediate backup run: %w", err)
		}
	case isScheduledOneTime(normalized.Schedule):
		if err := s.scheduleSyncer.DeleteJobSchedule(ctx, job.ID); err != nil {
			return nil, fmt.Errorf("clear backup job schedule: %w", err)
		}
		if err := s.runStarter.StartRunAt(ctx, in.MemberID, job.ID, *normalized.Schedule.OneTime); err != nil {
			return nil, fmt.Errorf("start scheduled backup run: %w", err)
		}
	default:
		if err := s.syncSchedule(ctx, job); err != nil {
			return nil, err
		}
	}
	return toDetails(job), nil
}

func isImmediateOneTime(s ScheduleInput) bool {
	if s.OneTime != nil {
		return false
	}
	switch strings.TrimSpace(s.Type) {
	case "one_time":
		return true
	case "recurring":
		return false
	default:
		// Legacy: no typed schedule and no one_time timestamp → run once immediately.
		return s.IntervalSeconds == nil &&
			(s.Cron == nil || strings.TrimSpace(*s.Cron) == "")
	}
}

func isScheduledOneTime(s ScheduleInput) bool {
	if s.OneTime == nil {
		return false
	}
	switch strings.TrimSpace(s.Type) {
	case "recurring":
		return false
	case "one_time", "":
		return true
	default:
		return true
	}
}

func (s *Service) Get(memberID, id string) (*BackupJobDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	job, err := s.repo.FindActiveByID(strings.TrimSpace(id), memberID)
	if err != nil {
		if errors.Is(err, storage.ErrBackupJobNotFound) {
			return nil, ErrBackupJobNotFound
		}
		return nil, fmt.Errorf("get backup job: %w", err)
	}
	return toDetails(job), nil
}

func (s *Service) List(memberID string) ([]BackupJobDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	jobs, err := s.repo.ListActive(memberID)
	if err != nil {
		return nil, fmt.Errorf("list backup jobs: %w", err)
	}
	out := make([]BackupJobDetails, 0, len(jobs))
	for i := range jobs {
		out = append(out, *toDetails(&jobs[i]))
	}
	return out, nil
}

func (s *Service) Update(memberID string, in UpdateInput) (*BackupJobDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if strings.TrimSpace(in.ID) == "" {
		return nil, ErrBackupJobNotFound
	}
	normalized, err := normalizeAndValidateInput(CreateInput{
		MemberID:  memberID,
		LastRun:   in.LastRun,
		NextRun:   in.NextRun,
		StartAt:   in.StartAt,
		EndAt:     in.EndAt,
		Active:    in.Active,
		Schedule:  in.Schedule,
		JobConfig: in.JobConfig,
	})
	if err != nil {
		return nil, err
	}
	if err := s.validateJobReferences(memberID, normalized.JobConfig); err != nil {
		return nil, err
	}

	job, err := s.repo.FindActiveByID(strings.TrimSpace(in.ID), memberID)
	if err != nil {
		if errors.Is(err, storage.ErrBackupJobNotFound) {
			return nil, ErrBackupJobNotFound
		}
		return nil, fmt.Errorf("find backup job: %w", err)
	}

	job.LastRun = normalized.LastRun
	job.NextRun = normalized.NextRun
	job.StartAt = normalized.StartAt
	job.EndAt = normalized.EndAt
	job.Active = normalized.Active
	job.ScheduleIntervalSeconds = normalized.Schedule.IntervalSeconds
	job.ScheduleCron = normalized.Schedule.Cron
	job.ScheduleOneTime = normalized.Schedule.OneTime
	job.OrganizationID = normalized.JobConfig.OrganizationID
	job.BucketStoreID = normalized.JobConfig.BucketStoreID
	job.SharePointSite = normalized.JobConfig.SharePointSite
	job.FilterDocumentLibraries = normalized.JobConfig.Filters.DocumentLibrariesCSV
	job.FilterMinFileSize = normalized.JobConfig.Filters.MinFileSize
	job.FilterMaxFileSize = normalized.JobConfig.Filters.MaxFileSize
	job.FilterCreatedAfter = normalized.JobConfig.Filters.CreatedAfter
	job.FilterUpdatedAfter = normalized.JobConfig.Filters.UpdatedAfter
	job.FilterCreatedBefore = normalized.JobConfig.Filters.CreatedBefore
	job.FilterUpdatedBefore = normalized.JobConfig.Filters.UpdatedBefore
	job.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(job); err != nil {
		if errors.Is(err, storage.ErrBackupJobNotFound) {
			return nil, ErrBackupJobNotFound
		}
		return nil, fmt.Errorf("update backup job: %w", err)
	}
	if err := s.syncSchedule(context.Background(), job); err != nil {
		return nil, err
	}
	return toDetails(job), nil
}

func (s *Service) Delete(memberID, id string) error {
	if strings.TrimSpace(memberID) == "" {
		return ErrInvalidMemberID
	}
	jobID := strings.TrimSpace(id)
	if err := s.repo.MarkInactive(jobID, memberID); err != nil {
		if errors.Is(err, storage.ErrBackupJobNotFound) {
			return ErrBackupJobNotFound
		}
		return fmt.Errorf("delete backup job: %w", err)
	}
	if err := s.scheduleSyncer.DeleteJobSchedule(context.Background(), jobID); err != nil {
		return fmt.Errorf("delete backup job schedule: %w", err)
	}
	return nil
}

func (s *Service) syncSchedule(ctx context.Context, job *storage.BackupJob) error {
	if usesRunStarterSchedule(job) {
		if err := s.scheduleSyncer.DeleteJobSchedule(ctx, job.ID); err != nil {
			return fmt.Errorf("sync backup job schedule: %w", err)
		}
		return nil
	}
	if err := s.scheduleSyncer.SyncJob(ctx, job); err != nil {
		return fmt.Errorf("sync backup job schedule: %w", err)
	}
	return nil
}

// usesRunStarterSchedule is true for one-time jobs executed via StartRun/StartRunAt (not Temporal schedules).
func usesRunStarterSchedule(job *storage.BackupJob) bool {
	if job.ScheduleOneTime != nil {
		return true
	}
	return job.ScheduleIntervalSeconds == nil &&
		(job.ScheduleCron == nil || strings.TrimSpace(*job.ScheduleCron) == "")
}

func (s *Service) validateJobReferences(memberID string, cfg JobConfigInput) error {
	if _, err := s.orgRepo.FindActiveByID(cfg.OrganizationID, memberID); err != nil {
		if errors.Is(err, storage.ErrOrganizationNotFound) {
			return ErrInvalidOrganizationID
		}
		return fmt.Errorf("validate organization reference: %w", err)
	}
	if _, err := s.bucketRepo.FindActiveByID(cfg.BucketStoreID, memberID); err != nil {
		if errors.Is(err, storage.ErrBucketStoreNotFound) {
			return ErrInvalidBucketStoreID
		}
		return fmt.Errorf("validate bucket store reference: %w", err)
	}
	return nil
}

func normalizeAndValidateInput(in CreateInput) (CreateInput, error) {
	in.JobConfig.OrganizationID = strings.TrimSpace(in.JobConfig.OrganizationID)
	in.JobConfig.BucketStoreID = strings.TrimSpace(in.JobConfig.BucketStoreID)
	in.JobConfig.SharePointSite = strings.TrimSpace(in.JobConfig.SharePointSite)
	in.JobConfig.Filters.DocumentLibrariesCSV = normalizeCSV(in.JobConfig.Filters.DocumentLibrariesCSV)
	if err := validateInput(in); err != nil {
		return CreateInput{}, err
	}
	return in, nil
}

func validateInput(in CreateInput) error {
	if in.StartAt != nil && in.EndAt != nil && in.StartAt.After(*in.EndAt) {
		return ErrInvalidStartAt
	}
	if strings.TrimSpace(in.JobConfig.OrganizationID) == "" {
		return ErrInvalidOrganizationID
	}
	if strings.TrimSpace(in.JobConfig.BucketStoreID) == "" {
		return ErrInvalidBucketStoreID
	}
	if strings.TrimSpace(in.JobConfig.SharePointSite) == "" {
		return ErrInvalidSharePointSite
	}

	if err := validateSchedule(in.Schedule); err != nil {
		return err
	}
	return validateFilters(in.JobConfig.Filters)
}

func validateSchedule(s ScheduleInput) error {
	scheduleType := strings.TrimSpace(s.Type)
	if scheduleType == "recurring" {
		if s.IntervalSeconds == nil || *s.IntervalSeconds <= 0 {
			return ErrInvalidInterval
		}
		return nil
	}
	if scheduleType == "one_time" {
		if s.OneTime != nil && !s.OneTime.After(time.Now().UTC()) {
			return ErrInvalidOneTime
		}
		return nil
	}

	// Backward compatibility: exactly one of interval, cron, or one_time.
	setCount := 0
	if s.IntervalSeconds != nil {
		setCount++
	}
	if s.Cron != nil && strings.TrimSpace(*s.Cron) != "" {
		setCount++
	}
	if s.OneTime != nil {
		setCount++
	}
	if setCount != 1 {
		return ErrInvalidSchedule
	}
	if s.IntervalSeconds != nil && *s.IntervalSeconds <= 0 {
		return ErrInvalidInterval
	}
	if s.Cron != nil {
		cron := strings.TrimSpace(*s.Cron)
		if cron == "" {
			return ErrInvalidCron
		}
		parts := strings.Fields(cron)
		if len(parts) < 5 || len(parts) > 6 {
			return ErrInvalidCron
		}
	}
	if s.OneTime != nil && !s.OneTime.After(time.Now().UTC()) {
		return ErrInvalidOneTime
	}
	return nil
}

func validateFilters(f FilterInput) error {
	if strings.Contains(f.DocumentLibrariesCSV, ",,") {
		return ErrInvalidDocumentLibrary
	}
	parts := strings.Split(f.DocumentLibrariesCSV, ",")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" && strings.TrimSpace(f.DocumentLibrariesCSV) != "" {
			return ErrInvalidDocumentLibrary
		}
	}

	if f.MinFileSize != nil && *f.MinFileSize < 0 {
		return ErrInvalidMinFileSize
	}
	if f.MaxFileSize != nil && *f.MaxFileSize < 0 {
		return ErrInvalidMaxFileSize
	}
	if f.MinFileSize != nil && f.MaxFileSize != nil && *f.MaxFileSize < *f.MinFileSize {
		return ErrInvalidFileSizeRange
	}
	if f.CreatedAfter != nil && f.CreatedBefore != nil && !f.CreatedBefore.After(*f.CreatedAfter) {
		return ErrInvalidCreatedRange
	}
	if f.UpdatedAfter != nil && f.UpdatedBefore != nil && !f.UpdatedBefore.After(*f.UpdatedAfter) {
		return ErrInvalidUpdatedRange
	}
	return nil
}

func normalizeCSV(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return strings.Join(out, ",")
}

func toDetails(job *storage.BackupJob) *BackupJobDetails {
	return &BackupJobDetails{
		ID:      job.ID,
		LastRun: job.LastRun,
		NextRun: job.NextRun,
		StartAt: job.StartAt,
		EndAt:   job.EndAt,
		Active:  job.Active,
		Schedule: ScheduleDetails{
			IntervalSeconds: job.ScheduleIntervalSeconds,
			Cron:            job.ScheduleCron,
			OneTime:         job.ScheduleOneTime,
		},
		JobConfig: JobConfigDetails{
			OrganizationID: job.OrganizationID,
			BucketStoreID:  job.BucketStoreID,
			SharePointSite: job.SharePointSite,
			Filters: FilterDetails{
				DocumentLibrariesCSV: job.FilterDocumentLibraries,
				MinFileSize:          job.FilterMinFileSize,
				MaxFileSize:          job.FilterMaxFileSize,
				CreatedAfter:         job.FilterCreatedAfter,
				UpdatedAfter:         job.FilterUpdatedAfter,
				CreatedBefore:        job.FilterCreatedBefore,
				UpdatedBefore:        job.FilterUpdatedBefore,
			},
		},
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}
