package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"

	"spsyncworker/internal/activities"
	"spsyncworker/internal/bootstrap"
	"spsyncworker/internal/config"
	"spsyncworker/internal/repository"
	temporalclient "spsyncworker/internal/temporal"
	"spsyncworker/internal/workflows"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Warn("Failed to load .env file", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	tClient, err := temporalclient.New(cfg.TemporalHostPort, cfg.TemporalNamespace)
	if err != nil {
		logger.Error("Failed to connect to Temporal", "error", err)
		os.Exit(1)
	}
	defer tClient.Close()

	repo := repository.NewJobsRepository(dbPool)

	w := worker.New(tClient, cfg.TemporalTaskQueue, worker.Options{})
	jobActivities := activities.New(logger, repo)
	w.RegisterWorkflow(workflows.JobExecutionWorkflow)
	w.RegisterActivityWithOptions(jobActivities.LogJobConfigActivity, activity.RegisterOptions{Name: "LogJobConfigActivity"})
	w.RegisterActivityWithOptions(jobActivities.SimulateRunActivity, activity.RegisterOptions{Name: "SimulateRunActivity"})

	if err := w.Start(); err != nil {
		logger.Error("Failed to start Temporal worker", "error", err)
		os.Exit(1)
	}
	defer w.Stop()

	reconciler := bootstrap.NewReconciler(logger, repo, tClient, cfg.TemporalTaskQueue)
	go reconciler.RunForever(ctx, time.Duration(cfg.PollIntervalSec)*time.Second)

	logger.Info("Temporal worker started",
		"namespace", cfg.TemporalNamespace,
		"taskQueue", cfg.TemporalTaskQueue,
		"hostPort", cfg.TemporalHostPort,
		"reconcileIntervalSec", cfg.PollIntervalSec,
	)

	<-ctx.Done()
	logger.Info("Shutdown signal received, stopping worker")
}
