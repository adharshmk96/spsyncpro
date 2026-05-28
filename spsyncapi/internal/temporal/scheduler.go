package temporal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"spsyncapi/internal/config"
	"spsyncapi/internal/storage"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// ScheduleOrchestrator syncs backup jobs to Temporal schedules.
type ScheduleOrchestrator struct {
	client    client.Client
	taskQueue string
	logger    *slog.Logger
}

// NewScheduleOrchestrator constructs a ScheduleOrchestrator.
func NewScheduleOrchestrator(c client.Client, cfg config.TemporalConfig, logger *slog.Logger) *ScheduleOrchestrator {
	return &ScheduleOrchestrator{
		client:    c,
		taskQueue: cfg.TaskQueue,
		logger:    logger,
	}
}

// SyncJob creates, updates, or deletes the Temporal schedule for a backup job.
func (o *ScheduleOrchestrator) SyncJob(ctx context.Context, job *storage.BackupJob) error {
	scheduleID := BackupScheduleID(job.ID)
	handle := o.client.ScheduleClient().GetHandle(ctx, scheduleID)

	if !job.Active {
		return o.deleteSchedule(ctx, handle, scheduleID)
	}

	spec, oneTime, err := buildScheduleSpec(job)
	if err != nil {
		return err
	}

	action := &client.ScheduleWorkflowAction{
		Workflow:  ScheduledBackupWorkflow,
		Args:      []interface{}{ScheduledBackupInput{JobID: job.ID, MemberID: job.MemberID}},
		TaskQueue: o.taskQueue,
	}

	_, err = handle.Describe(ctx)
	if err != nil {
		opts := client.ScheduleOptions{
			ID:     scheduleID,
			Spec:   spec,
			Action: action,
			Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
		}
		if oneTime {
			opts.RemainingActions = 1
		}
		_, createErr := o.client.ScheduleClient().Create(ctx, opts)
		if createErr != nil {
			return fmt.Errorf("schedule orchestrator: create %s: %w", scheduleID, createErr)
		}
		o.logger.Info("temporal schedule created", "schedule_id", scheduleID, "job_id", job.ID)
		return nil
	}

	updateErr := handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			sched := input.Description.Schedule
			sched.Spec = &spec
			sched.Action = action
			return &client.ScheduleUpdate{Schedule: &sched}, nil
		},
	})
	if updateErr != nil {
		return fmt.Errorf("schedule orchestrator: update %s: %w", scheduleID, updateErr)
	}
	o.logger.Info("temporal schedule updated", "schedule_id", scheduleID, "job_id", job.ID)
	return nil
}

// DeleteJobSchedule removes the Temporal schedule for a backup job.
func (o *ScheduleOrchestrator) DeleteJobSchedule(ctx context.Context, jobID string) error {
	scheduleID := BackupScheduleID(jobID)
	handle := o.client.ScheduleClient().GetHandle(ctx, scheduleID)
	return o.deleteSchedule(ctx, handle, scheduleID)
}

func (o *ScheduleOrchestrator) deleteSchedule(ctx context.Context, handle client.ScheduleHandle, scheduleID string) error {
	_, err := handle.Describe(ctx)
	if err != nil {
		return nil
	}
	if err := handle.Delete(ctx); err != nil {
		return fmt.Errorf("schedule orchestrator: delete %s: %w", scheduleID, err)
	}
	o.logger.Info("temporal schedule deleted", "schedule_id", scheduleID)
	return nil
}

func buildScheduleSpec(job *storage.BackupJob) (client.ScheduleSpec, bool, error) {
	spec := client.ScheduleSpec{}
	oneTime := false

	if job.ScheduleIntervalSeconds != nil && *job.ScheduleIntervalSeconds > 0 {
		d := time.Duration(*job.ScheduleIntervalSeconds) * time.Second
		spec.Intervals = []client.ScheduleIntervalSpec{{Every: d}}
	} else if job.ScheduleCron != nil && *job.ScheduleCron != "" {
		spec.CronExpressions = []string{*job.ScheduleCron}
	} else if job.ScheduleOneTime != nil {
		oneTime = true
		t := job.ScheduleOneTime.UTC()
		spec.StartAt = t
		spec.Calendars = []client.ScheduleCalendarSpec{
			{
				Year:       []client.ScheduleRange{{Start: t.Year()}},
				Month:      []client.ScheduleRange{{Start: int(t.Month())}},
				DayOfMonth: []client.ScheduleRange{{Start: t.Day()}},
				Hour:       []client.ScheduleRange{{Start: t.Hour()}},
				Minute:     []client.ScheduleRange{{Start: t.Minute()}},
				Second:     []client.ScheduleRange{{Start: t.Second()}},
			},
		}
	} else {
		return client.ScheduleSpec{}, false, fmt.Errorf("backup job %s has no schedule", job.ID)
	}

	if job.StartAt != nil {
		if spec.StartAt.IsZero() || job.StartAt.After(spec.StartAt) {
			spec.StartAt = job.StartAt.UTC()
		}
	}
	if job.EndAt != nil {
		spec.EndAt = job.EndAt.UTC()
	}

	return spec, oneTime, nil
}

// NoopScheduleSyncer is used in unit tests when Temporal is not available.
type NoopScheduleSyncer struct{}

func (NoopScheduleSyncer) SyncJob(context.Context, *storage.BackupJob) error { return nil }

func (NoopScheduleSyncer) DeleteJobSchedule(context.Context, string) error { return nil }

// ScheduleSyncer is implemented by ScheduleOrchestrator.
type ScheduleSyncer interface {
	SyncJob(ctx context.Context, job *storage.BackupJob) error
	DeleteJobSchedule(ctx context.Context, jobID string) error
}

var _ ScheduleSyncer = (*ScheduleOrchestrator)(nil)
