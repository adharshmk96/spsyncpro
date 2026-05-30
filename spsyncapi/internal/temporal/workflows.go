package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	defaultMaxConcurrentTransfers = 5
	fetchMetadataTimeout          = 30 * time.Minute
	transferActivityTimeout       = 2 * time.Hour
	transferHeartbeatTimeout      = 60 * time.Second
)

// BackupRunWorkflow orchestrates a single backup run (manual, scheduled, or resumed).
func BackupRunWorkflow(ctx workflow.Context, in RunWorkflowInput) error {
	return runTransferWorkflow(ctx, in)
}

// RestoreRunWorkflow orchestrates a single restore run.
func RestoreRunWorkflow(ctx workflow.Context, in RunWorkflowInput) error {
	return runTransferWorkflow(ctx, in)
}

// ScheduledBackupWorkflow is started by a Temporal schedule; creates a run then transfers files.
func ScheduledBackupWorkflow(ctx workflow.Context, in ScheduledBackupInput) error {
	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})

	var runID string
	if err := workflow.ExecuteActivity(actCtx, "CreateBackupRun", in).Get(ctx, &runID); err != nil {
		return err
	}

	return runTransferWorkflow(ctx, RunWorkflowInput{
		RunID:                  runID,
		JobID:                  in.JobID,
		MemberID:               in.MemberID,
		Kind:                   RunKindBackup,
		Resume:                 true,
		MaxConcurrentTransfers: in.MaxConcurrentTransfers,
	})
}

func runTransferWorkflow(ctx workflow.Context, in RunWorkflowInput) error {
	if !in.Resume {
		actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
		})
		switch in.Kind {
		case RunKindBackup:
			var runID string
			if err := workflow.ExecuteActivity(actCtx, "CreateBackupRun", ScheduledBackupInput{
				JobID:                  in.JobID,
				MemberID:               in.MemberID,
				MaxConcurrentTransfers: in.MaxConcurrentTransfers,
			}).Get(ctx, &runID); err != nil {
				return err
			}
			in.RunID = runID
		case RunKindRestore:
			var runID string
			if err := workflow.ExecuteActivity(actCtx, "CreateRestoreRun", ScheduledBackupInput{
				JobID:    in.JobID,
				MemberID: in.MemberID,
			}).Get(ctx, &runID); err != nil {
				return err
			}
			in.RunID = runID
		}
	}

	runInput := FinalizeRunInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	}

	fetchCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: fetchMetadataTimeout,
		HeartbeatTimeout:    transferHeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	var meta FetchFileMetadataOutput
	if err := workflow.ExecuteActivity(fetchCtx, fetchFileMetadataActivityName, FetchFileMetadataInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	}).Get(ctx, &meta); err != nil {
		return err
	}

	transferCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: transferActivityTimeout,
		HeartbeatTimeout:    transferHeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	maxConcurrent := effectiveMaxConcurrentTransfers(in.MaxConcurrentTransfers)
	for i := 0; i < len(meta.Files); i += maxConcurrent {
		end := i + maxConcurrent
		if end > len(meta.Files) {
			end = len(meta.Files)
		}
		batch := meta.Files[i:end]

		futures := make([]workflow.Future, len(batch))
		for j, file := range batch {
			futures[j] = workflow.ExecuteActivity(transferCtx, transferSingleFileActivityName, TransferSingleFileInput{
				RunID:    in.RunID,
				JobID:    in.JobID,
				MemberID: in.MemberID,
				Kind:     in.Kind,
				File:     file,
			})
		}
		for _, f := range futures {
			if err := f.Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	finalizeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	return workflow.ExecuteActivity(finalizeCtx, finalizeRunActivityName, runInput).Get(ctx, nil)
}
