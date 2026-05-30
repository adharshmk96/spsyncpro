package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"spsyncapi/internal/config"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// RunExecutor starts and stops Temporal workflows for backup/restore runs.
type RunExecutor struct {
	client                 client.Client
	taskQueue              string
	maxConcurrentTransfers int
}

// NewRunExecutor constructs a RunExecutor.
func NewRunExecutor(c client.Client, cfg config.TemporalConfig) *RunExecutor {
	return &RunExecutor{
		client:                 c,
		taskQueue:              cfg.TaskQueue,
		maxConcurrentTransfers: cfg.MaxConcurrentTransfers,
	}
}

// StartBackupRun starts a backup run workflow for an existing run row.
func (e *RunExecutor) StartBackupRun(ctx context.Context, in RunWorkflowInput) error {
	in.Kind = RunKindBackup
	in.Resume = true
	return e.startRun(ctx, BackupWorkflowID(in.RunID), BackupRunWorkflow, in, 0)
}

// StartBackupRunAt starts a backup run workflow at or after the given time.
func (e *RunExecutor) StartBackupRunAt(ctx context.Context, in RunWorkflowInput, at time.Time) error {
	in.Kind = RunKindBackup
	in.Resume = true
	delay := time.Until(at.UTC())
	if delay < 0 {
		delay = 0
	}
	return e.startRun(ctx, BackupWorkflowID(in.RunID), BackupRunWorkflow, in, delay)
}

// StartRestoreRun starts a restore run workflow for an existing run row.
func (e *RunExecutor) StartRestoreRun(ctx context.Context, in RunWorkflowInput) error {
	in.Kind = RunKindRestore
	in.Resume = true
	return e.startRun(ctx, RestoreWorkflowID(in.RunID), RestoreRunWorkflow, in, 0)
}

// StartRestoreRunAt starts a restore run workflow at or after the given time.
func (e *RunExecutor) StartRestoreRunAt(ctx context.Context, in RunWorkflowInput, at time.Time) error {
	in.Kind = RunKindRestore
	in.Resume = true
	delay := time.Until(at.UTC())
	if delay < 0 {
		delay = 0
	}
	return e.startRun(ctx, RestoreWorkflowID(in.RunID), RestoreRunWorkflow, in, delay)
}

func (e *RunExecutor) startRun(ctx context.Context, workflowID string, workflow interface{}, in RunWorkflowInput, startDelay time.Duration) error {
	if in.MaxConcurrentTransfers == 0 {
		in.MaxConcurrentTransfers = e.maxConcurrentTransfers
	}
	opts := client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             e.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}
	if startDelay > 0 {
		opts.StartDelay = startDelay
	}
	_, err := e.client.ExecuteWorkflow(ctx, opts, workflow, in)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return nil
		}
		return fmt.Errorf("start workflow %s: %w", workflowID, err)
	}
	return nil
}

// StopBackupRun requests cancellation of a backup run workflow.
func (e *RunExecutor) StopBackupRun(ctx context.Context, runID string) error {
	return e.client.CancelWorkflow(ctx, BackupWorkflowID(runID), "")
}

// StopRestoreRun requests cancellation of a restore run workflow.
func (e *RunExecutor) StopRestoreRun(ctx context.Context, runID string) error {
	return e.client.CancelWorkflow(ctx, RestoreWorkflowID(runID), "")
}

// ResumeBackupRunIfNeeded starts a workflow for an incomplete backup run when not already running.
func (e *RunExecutor) ResumeBackupRunIfNeeded(ctx context.Context, in RunWorkflowInput) error {
	workflowID := BackupWorkflowID(in.RunID)
	desc, err := e.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err == nil {
		if desc.WorkflowExecutionInfo.Status == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
			return nil
		}
		if desc.WorkflowExecutionInfo.Status == enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
			return nil
		}
	}
	in.Kind = RunKindBackup
	in.Resume = true
	if in.MaxConcurrentTransfers == 0 {
		in.MaxConcurrentTransfers = e.maxConcurrentTransfers
	}
	_, err = e.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             e.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	}, BackupRunWorkflow, in)
	if err != nil {
		return fmt.Errorf("resume backup workflow %s: %w", workflowID, err)
	}
	return nil
}

// ResumeRestoreRunIfNeeded starts a workflow for an incomplete restore run when not already running.
func (e *RunExecutor) ResumeRestoreRunIfNeeded(ctx context.Context, in RunWorkflowInput) error {
	workflowID := RestoreWorkflowID(in.RunID)
	desc, err := e.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err == nil {
		if desc.WorkflowExecutionInfo.Status == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
			return nil
		}
		if desc.WorkflowExecutionInfo.Status == enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
			return nil
		}
	}
	in.Kind = RunKindRestore
	in.Resume = true
	if in.MaxConcurrentTransfers == 0 {
		in.MaxConcurrentTransfers = e.maxConcurrentTransfers
	}
	_, err = e.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             e.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	}, RestoreRunWorkflow, in)
	if err != nil {
		return fmt.Errorf("resume restore workflow %s: %w", workflowID, err)
	}
	return nil
}
