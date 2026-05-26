package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"spsyncapi/internal/config"
	"spsyncapi/internal/server"
	"spsyncapi/internal/telemetry"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start the SPSync API HTTP server and listen for incoming requests.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.SlogLevel(),
		}))

		metrics, err := telemetry.Init(cmd.Context(), cfg, logger)
		if err != nil {
			return fmt.Errorf("init metrics: %w", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			if err := metrics.Shutdown(shutdownCtx); err != nil {
				logger.Error("metrics shutdown failed", "error", err)
			}
		}()

		srv, err := server.New(cfg, logger, metrics)
		if err != nil {
			return fmt.Errorf("create server: %w", err)
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		logger.Info("starting server", "address", cfg.Address())

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Start()
		}()

		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("shutdown server: %w", err)
			}

			logger.Info("server stopped gracefully")
			return nil
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("run server: %w", err)
			}
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
