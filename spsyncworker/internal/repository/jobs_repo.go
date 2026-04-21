package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

type Job struct {
	ID                  string
	RunMode             string
	Recurrence          *string
	StartAt             *time.Time
	NextRunAt           *time.Time
	LastRunAt           *time.Time
	StorageConfig       json.RawMessage
	FilterConfig        json.RawMessage
	DocumentLibraryList []string
	RunnerMetadata      json.RawMessage
	Status              string
}

type JobsRepository struct {
	pool *pgxpool.Pool
}

func NewJobsRepository(pool *pgxpool.Pool) *JobsRepository {
	return &JobsRepository{pool: pool}
}

func (r *JobsRepository) ListSchedulableActiveJobs(ctx context.Context) ([]Job, error) {
	const query = `
		SELECT
			id,
			run_mode,
			recurrence,
			start_at,
			next_run_at,
			last_run_at,
			storage_config,
			filter_config,
			document_library_list,
			runner_metadata,
			status
		FROM backupjobs
		WHERE status = 'ACTIVE'
		  AND (
				(
					(next_run_at IS NULL OR next_run_at <= NOW())
					AND (run_mode <> 'ONE_TIME_AT' OR last_run_at IS NULL)
				)
				OR COALESCE(runner_metadata->>'manualRunAt', '') <> ''
			)
		ORDER BY created_at ASC;
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]Job, 0)
	for rows.Next() {
		var job Job
		if err := rows.Scan(
			&job.ID,
			&job.RunMode,
			&job.Recurrence,
			&job.StartAt,
			&job.NextRunAt,
			&job.LastRunAt,
			&job.StorageConfig,
			&job.FilterConfig,
			&job.DocumentLibraryList,
			&job.RunnerMetadata,
			&job.Status,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active job row: %w", err)
		}

		if err := validateJob(job); err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("active job rows iteration failed: %w", err)
	}

	return jobs, nil
}

// ManualRunPending reports whether the UI requested an on-demand run via runner_metadata.manualRunAt.
func (j Job) ManualRunPending() bool {
	var meta map[string]any
	if err := json.Unmarshal(j.RunnerMetadata, &meta); err != nil || meta == nil {
		return false
	}
	raw, ok := meta["manualRunAt"]
	if !ok || raw == nil {
		return false
	}
	s, ok := raw.(string)
	return ok && strings.TrimSpace(s) != ""
}

func (r *JobsRepository) ClearManualRunAt(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("jobID is required")
	}

	const q = `
		UPDATE backupjobs
		SET
			runner_metadata = COALESCE(runner_metadata, '{}'::jsonb) - 'manualRunAt',
			updated_at = NOW()
		WHERE id = $1
	`
	if _, err := r.pool.Exec(ctx, q, jobID); err != nil {
		return fmt.Errorf("failed to clear manualRunAt: %w", err)
	}
	return nil
}

func (r *JobsRepository) RecordSuccessfulRun(
	ctx context.Context,
	jobID string,
	scheduledFor time.Time,
	startedAt time.Time,
	finishedAt time.Time,
) error {
	if jobID == "" {
		return fmt.Errorf("jobID is required")
	}

	runID := uuid.NewString()
	now := time.Now().UTC()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin run transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertRunQuery = `
		INSERT INTO backup_job_runs (
			id,
			job_id,
			scheduled_for,
			started_at,
			finished_at,
			status,
			attempt,
			runner_id,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'SUCCESS', 1, 'temporal-worker', $6, $6)
		ON CONFLICT (job_id, scheduled_for) DO NOTHING;
	`

	if _, err := tx.Exec(ctx, insertRunQuery, runID, jobID, scheduledFor, startedAt, finishedAt, now); err != nil {
		return fmt.Errorf("failed to insert backup_job_runs success record: %w", err)
	}

	const updateJobQuery = `
		UPDATE backupjobs
		SET
			last_run_at = $1,
			last_run_status = 'SUCCESS',
			updated_at = $1
		WHERE id = $2;
	`

	if _, err := tx.Exec(ctx, updateJobQuery, finishedAt, jobID); err != nil {
		return fmt.Errorf("failed to update backupjobs last run fields: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit run transaction: %w", err)
	}

	return nil
}

func validateJob(job Job) error {
	if job.ID == "" {
		return fmt.Errorf("job id is required")
	}

	switch job.RunMode {
	case "IMMEDIATE":
	case "ONE_TIME_AT":
		if job.StartAt == nil {
			return fmt.Errorf("job %s missing start_at for ONE_TIME_AT", job.ID)
		}
	case "RECURRING":
		if job.StartAt == nil {
			return fmt.Errorf("job %s missing start_at for RECURRING", job.ID)
		}
		if job.Recurrence == nil || (*job.Recurrence != "DAILY" && *job.Recurrence != "WEEKLY" && *job.Recurrence != "MONTHLY") {
			return fmt.Errorf("job %s has invalid recurrence", job.ID)
		}
	default:
		return fmt.Errorf("job %s has invalid run mode: %s", job.ID, job.RunMode)
	}

	var storage map[string]any
	if err := json.Unmarshal(job.StorageConfig, &storage); err != nil {
		return fmt.Errorf("job %s has invalid storage_config JSON: %w", job.ID, err)
	}

	var filter map[string]any
	if err := json.Unmarshal(job.FilterConfig, &filter); err != nil {
		return fmt.Errorf("job %s has invalid filter_config JSON: %w", job.ID, err)
	}

	return nil
}
