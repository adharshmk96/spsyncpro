package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const transferActivityName = "TransferFiles"

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
		RunID:    runID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     RunKindBackup,
		Resume:   true,
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
				JobID:    in.JobID,
				MemberID: in.MemberID,
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

	transferCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	return workflow.ExecuteActivity(transferCtx, transferActivityName, TransferFilesInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	}).Get(ctx, nil)
}
