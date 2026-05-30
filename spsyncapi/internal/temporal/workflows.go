package temporal

import (
	"fmt"
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

	return BackupRunWorkflow(ctx, RunWorkflowInput{
		RunID:                  runID,
		JobID:                  in.JobID,
		MemberID:               in.MemberID,
		Kind:                   RunKindBackup,
		MaxConcurrentTransfers: in.MaxConcurrentTransfers,
	})
}

func runTransferWorkflow(ctx workflow.Context, in RunWorkflowInput) error {
	if in.RunID == "" {
		return fmt.Errorf("run transfer workflow: run_id is required")
	}

	runInput := FinalizeRunInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	}

	// 1. metadata sync
	if err := syncAllFileMetadata(ctx, in); err != nil {
		return err
	}

	// 2. transfer pending files
	maxConcurrent := effectiveMaxConcurrentTransfers(in.MaxConcurrentTransfers)
	if err := transferAllPendingFiles(ctx, in, maxConcurrent); err != nil {
		return err
	}

	// 3. finalize
	finalizeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})
	return workflow.ExecuteActivity(finalizeCtx, finalizeRunActivityName, runInput).Get(ctx, nil)
}

func syncAllFileMetadata(ctx workflow.Context, in RunWorkflowInput) error {
	fetchCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: fetchMetadataTimeout,
		HeartbeatTimeout:    transferHeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	syncInput := SyncFileMetadataPageInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	}
	for {
		var syncOut SyncFileMetadataPageOutput
		if err := workflow.ExecuteActivity(fetchCtx, syncFileMetadataPageActivityName, syncInput).Get(ctx, &syncOut); err != nil {
			return err
		}
		if syncOut.Complete {
			return nil
		}
	}
}

func transferAllPendingFiles(ctx workflow.Context, in RunWorkflowInput, maxConcurrent int) error {
	transferCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: transferActivityTimeout,
		HeartbeatTimeout:    transferHeartbeatTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	listCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumAttempts: 3,
		},
	})

	for {
		var pending ListPendingFileLogsOutput
		if err := workflow.ExecuteActivity(listCtx, listPendingFileLogsActivityName, ListPendingFileLogsInput{
			RunID:    in.RunID,
			JobID:    in.JobID,
			MemberID: in.MemberID,
			Kind:     in.Kind,
			Limit:    listPendingFileLogsBatchSize,
		}).Get(ctx, &pending); err != nil {
			return err
		}
		if len(pending.Files) == 0 {
			return nil
		}

		for i := 0; i < len(pending.Files); i += maxConcurrent {
			end := i + maxConcurrent
			if end > len(pending.Files) {
				end = len(pending.Files)
			}
			batch := pending.Files[i:end]

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
	}
}
