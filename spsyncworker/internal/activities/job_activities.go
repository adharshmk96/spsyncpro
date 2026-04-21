package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"spsyncworker/internal/repository"
	"spsyncworker/internal/workflows"
)

type JobActivities struct {
	logger *slog.Logger
	repo   *repository.JobsRepository
}

func New(logger *slog.Logger, repo *repository.JobsRepository) *JobActivities {
	return &JobActivities{
		logger: logger,
		repo:   repo,
	}
}

func (a *JobActivities) LogJobConfigActivity(_ context.Context, input workflows.JobActivityInput) error {
	a.logger.Info(
		"Job configuration loaded",
		"jobID", input.JobID,
		"runMode", input.RunMode,
		"recurrence", input.Recurrence,
		"storageConfig", redactJSON(input.StorageConfigJSON),
		"filterConfig", input.FilterConfigJSON,
	)
	return nil
}

func (a *JobActivities) SimulateRunActivity(ctx context.Context, input workflows.JobActivityInput) error {
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(2 * time.Second)

	if err := a.repo.RecordSuccessfulRun(ctx, input.JobID, startedAt, startedAt, finishedAt); err != nil {
		return fmt.Errorf("failed to persist successful run: %w", err)
	}

	a.logger.Info(
		"Simulated backup run",
		"jobID", input.JobID,
		"startedAt", startedAt.Format(time.RFC3339),
		"finishedAt", finishedAt.Format(time.RFC3339),
	)
	return nil
}

func redactJSON(raw string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "{}"
	}

	for _, key := range []string{"connectionString", "accessKeyID", "secretAccessKey"} {
		if _, exists := payload[key]; exists {
			payload[key] = "***REDACTED***"
		}
	}

	// Redact nested storage provider blocks.
	for _, nested := range []string{"azureBlobConfig", "awsS3Config"} {
		if obj, ok := payload[nested].(map[string]any); ok {
			for _, key := range []string{"connectionString", "accessKeyID", "secretAccessKey"} {
				if _, exists := obj[key]; exists {
					obj[key] = "***REDACTED***"
				}
			}
			payload[nested] = obj
		}
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}
