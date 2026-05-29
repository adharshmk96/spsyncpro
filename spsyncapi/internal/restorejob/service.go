package restorejob

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
	ErrRestoreJobNotFound      = errors.New("restore job not found")
	ErrInvalidStartAt          = errors.New("start_at must be in the future when set")
	ErrInvalidStartAtPast      = errors.New("start_at cannot be in the past")
	ErrInvalidOrganizationID   = errors.New("job_config.organization is required")
	ErrInvalidBucketStoreID    = errors.New("job_config.bucket_store is required")
	ErrInvalidSharePointSite   = errors.New("job_config.share_point_site is required")
	ErrInvalidMemberID         = errors.New("member id is required")
)

type JobConfigInput struct {
	OrganizationID string
	BucketStoreID  string
	SharePointSite string
}

type CreateInput struct {
	MemberID  string
	StartAt   *time.Time
	LastRun   *time.Time
	Active    bool
	JobConfig JobConfigInput
}

type UpdateInput struct {
	ID        string
	StartAt   *time.Time
	LastRun   *time.Time
	Active    bool
	JobConfig JobConfigInput
}

type JobConfigDetails struct {
	OrganizationID string `json:"organization"`
	BucketStoreID  string `json:"bucket_store"`
	SharePointSite string `json:"share_point_site"`
}

type RestoreJobDetails struct {
	ID        string           `json:"id"`
	StartAt   *time.Time       `json:"start_at,omitempty"`
	LastRun   *time.Time       `json:"last_run,omitempty"`
	Active    bool             `json:"active"`
	JobConfig JobConfigDetails `json:"job_config"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// RunStarter starts restore runs for a job.
type RunStarter interface {
	StartRun(ctx context.Context, memberID, jobID string) error
	StartRunAt(ctx context.Context, memberID, jobID string, at time.Time) error
}

type Service struct {
	repo       *storage.RestoreJobRepository
	orgRepo    *storage.OrganizationRepository
	bucketRepo *storage.BucketStoreRepository
	runStarter RunStarter
	logger     *slog.Logger
}

type ServiceConfig struct {
	Repo       *storage.RestoreJobRepository
	OrgRepo    *storage.OrganizationRepository
	BucketRepo *storage.BucketStoreRepository
	RunStarter RunStarter
	Logger     *slog.Logger
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("restore job repo is required")
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
	starter := cfg.RunStarter
	if starter == nil {
		starter = noopRunStarter{}
	}
	return &Service{
		repo:       cfg.Repo,
		orgRepo:    cfg.OrgRepo,
		bucketRepo: cfg.BucketRepo,
		runStarter: starter,
		logger:     cfg.Logger,
	}, nil
}

type noopRunStarter struct{}

func (noopRunStarter) StartRun(context.Context, string, string) error { return nil }
func (noopRunStarter) StartRunAt(context.Context, string, string, time.Time) error {
	return nil
}

func (s *Service) Create(in CreateInput) (*RestoreJobDetails, error) {
	if strings.TrimSpace(in.MemberID) == "" {
		return nil, ErrInvalidMemberID
	}
	normalized, err := normalizeAndValidateCreate(in)
	if err != nil {
		return nil, err
	}
	if err := s.validateJobReferences(in.MemberID, normalized.JobConfig); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job := &storage.RestoreJob{
		ID:             uuid.NewString(),
		MemberID:       in.MemberID,
		StartAt:        normalized.StartAt,
		LastRun:        normalized.LastRun,
		Active:         normalized.Active,
		OrganizationID: normalized.JobConfig.OrganizationID,
		BucketStoreID:  normalized.JobConfig.BucketStoreID,
		SharePointSite: normalized.JobConfig.SharePointSite,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(job); err != nil {
		return nil, fmt.Errorf("create restore job: %w", err)
	}
	ctx := context.Background()
	if normalized.StartAt == nil {
		if err := s.runStarter.StartRun(ctx, in.MemberID, job.ID); err != nil {
			return nil, fmt.Errorf("start immediate restore run: %w", err)
		}
	} else {
		if err := s.runStarter.StartRunAt(ctx, in.MemberID, job.ID, *normalized.StartAt); err != nil {
			return nil, fmt.Errorf("start scheduled restore run: %w", err)
		}
	}
	return toDetails(job), nil
}

func (s *Service) Get(memberID, id string) (*RestoreJobDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	job, err := s.repo.FindActiveByID(strings.TrimSpace(id), memberID)
	if err != nil {
		if errors.Is(err, storage.ErrRestoreJobNotFound) {
			return nil, ErrRestoreJobNotFound
		}
		return nil, fmt.Errorf("get restore job: %w", err)
	}
	return toDetails(job), nil
}

func (s *Service) List(memberID string) ([]RestoreJobDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	jobs, err := s.repo.ListActive(memberID)
	if err != nil {
		return nil, fmt.Errorf("list restore jobs: %w", err)
	}
	out := make([]RestoreJobDetails, 0, len(jobs))
	for i := range jobs {
		out = append(out, *toDetails(&jobs[i]))
	}
	return out, nil
}

func (s *Service) Update(memberID string, in UpdateInput) (*RestoreJobDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if strings.TrimSpace(in.ID) == "" {
		return nil, ErrRestoreJobNotFound
	}
	normalized, err := normalizeAndValidateUpdate(in)
	if err != nil {
		return nil, err
	}
	if err := s.validateJobReferences(memberID, normalized.JobConfig); err != nil {
		return nil, err
	}

	job, err := s.repo.FindActiveByID(strings.TrimSpace(in.ID), memberID)
	if err != nil {
		if errors.Is(err, storage.ErrRestoreJobNotFound) {
			return nil, ErrRestoreJobNotFound
		}
		return nil, fmt.Errorf("find restore job: %w", err)
	}

	job.StartAt = normalized.StartAt
	job.LastRun = normalized.LastRun
	job.Active = normalized.Active
	job.OrganizationID = normalized.JobConfig.OrganizationID
	job.BucketStoreID = normalized.JobConfig.BucketStoreID
	job.SharePointSite = normalized.JobConfig.SharePointSite
	job.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(job); err != nil {
		if errors.Is(err, storage.ErrRestoreJobNotFound) {
			return nil, ErrRestoreJobNotFound
		}
		return nil, fmt.Errorf("update restore job: %w", err)
	}
	return toDetails(job), nil
}

func (s *Service) Delete(memberID, id string) error {
	if strings.TrimSpace(memberID) == "" {
		return ErrInvalidMemberID
	}
	if err := s.repo.MarkInactive(strings.TrimSpace(id), memberID); err != nil {
		if errors.Is(err, storage.ErrRestoreJobNotFound) {
			return ErrRestoreJobNotFound
		}
		return fmt.Errorf("delete restore job: %w", err)
	}
	return nil
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

func normalizeAndValidateCreate(in CreateInput) (CreateInput, error) {
	in.JobConfig.OrganizationID = strings.TrimSpace(in.JobConfig.OrganizationID)
	in.JobConfig.BucketStoreID = strings.TrimSpace(in.JobConfig.BucketStoreID)
	in.JobConfig.SharePointSite = strings.TrimSpace(in.JobConfig.SharePointSite)
	if strings.TrimSpace(in.JobConfig.OrganizationID) == "" {
		return CreateInput{}, ErrInvalidOrganizationID
	}
	if strings.TrimSpace(in.JobConfig.BucketStoreID) == "" {
		return CreateInput{}, ErrInvalidBucketStoreID
	}
	if strings.TrimSpace(in.JobConfig.SharePointSite) == "" {
		return CreateInput{}, ErrInvalidSharePointSite
	}
	if in.StartAt != nil {
		now := time.Now().UTC()
		if !in.StartAt.After(now) {
			return CreateInput{}, ErrInvalidStartAtPast
		}
	}
	return in, nil
}

func normalizeAndValidateUpdate(in UpdateInput) (UpdateInput, error) {
	out := UpdateInput{
		ID:        in.ID,
		StartAt:   in.StartAt,
		LastRun:   in.LastRun,
		Active:    in.Active,
		JobConfig: in.JobConfig,
	}
	out.JobConfig.OrganizationID = strings.TrimSpace(in.JobConfig.OrganizationID)
	out.JobConfig.BucketStoreID = strings.TrimSpace(in.JobConfig.BucketStoreID)
	out.JobConfig.SharePointSite = strings.TrimSpace(in.JobConfig.SharePointSite)
	if strings.TrimSpace(out.JobConfig.OrganizationID) == "" {
		return UpdateInput{}, ErrInvalidOrganizationID
	}
	if strings.TrimSpace(out.JobConfig.BucketStoreID) == "" {
		return UpdateInput{}, ErrInvalidBucketStoreID
	}
	if strings.TrimSpace(out.JobConfig.SharePointSite) == "" {
		return UpdateInput{}, ErrInvalidSharePointSite
	}
	return out, nil
}

func toDetails(job *storage.RestoreJob) *RestoreJobDetails {
	return &RestoreJobDetails{
		ID:      job.ID,
		StartAt: job.StartAt,
		LastRun: job.LastRun,
		Active:  job.Active,
		JobConfig: JobConfigDetails{
			OrganizationID: job.OrganizationID,
			BucketStoreID:  job.BucketStoreID,
			SharePointSite: job.SharePointSite,
		},
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}
