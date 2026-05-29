package backuprun_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/backuprun"
	"spsyncapi/internal/temporal"
)

type mockRunExecutor struct {
	started []temporal.RunWorkflowInput
	stopped []string
}

func (m *mockRunExecutor) StartBackupRun(_ context.Context, in temporal.RunWorkflowInput) error {
	m.started = append(m.started, in)
	return nil
}

func (m *mockRunExecutor) StartBackupRunAt(_ context.Context, in temporal.RunWorkflowInput, _ time.Time) error {
	m.started = append(m.started, in)
	return nil
}

func (m *mockRunExecutor) StopBackupRun(_ context.Context, runID string) error {
	m.stopped = append(m.stopped, runID)
	return nil
}

func TestStartAndStopRun(t *testing.T) {
	_, jobRepo, runRepo := newTestBackupRunService(t)
	exec := &mockRunExecutor{}
	svcWithExec, err := backuprun.NewService(backuprun.ServiceConfig{
		RunRepo:  runRepo,
		JobRepo:  jobRepo,
		Executor: exec,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}

	jobID := seedBackupJob(t, jobRepo)
	details, err := svcWithExec.StartRun(context.Background(), testMemberA, jobID)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if len(exec.started) != 1 || exec.started[0].RunID != details.ID {
		t.Fatalf("executor started: %+v", exec.started)
	}

	_, err = svcWithExec.StopRun(context.Background(), testMemberA, details.ID)
	if err != nil {
		t.Fatalf("StopRun: %v", err)
	}
	if len(exec.stopped) != 1 || exec.stopped[0] != details.ID {
		t.Fatalf("executor stopped: %+v", exec.stopped)
	}

	// Mark complete and stop should conflict.
	now := time.Now().UTC()
	run, _ := runRepo.FindByID(details.ID, testMemberA)
	run.EndAt = &now
	_ = runRepo.Update(run)
	_, err = svcWithExec.StopRun(context.Background(), testMemberA, details.ID)
	if err == nil {
		t.Fatal("expected error stopping completed run")
	}
}

func TestStartRunRequiresExecutor(t *testing.T) {
	svc, jobRepo, _ := newTestBackupRunService(t)
	jobID := seedBackupJob(t, jobRepo)
	_, err := svc.StartRun(context.Background(), testMemberA, jobID)
	if err == nil {
		t.Fatal("expected error without executor")
	}
}
