package temporal

import (
	"context"
	"log/slog"
	"time"
)

// RunReconcileLoop periodically reconciles DB state to Temporal and runs an extra pass after reconnect.
func RunReconcileLoop(ctx context.Context, deps ReconcileDeps, interval time.Duration, logger *slog.Logger) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	wasHealthy := TemporalHealthy(ctx, deps.TemporalClient)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthy := TemporalHealthy(ctx, deps.TemporalClient)
			if !healthy {
				if wasHealthy {
					logger.Warn("temporal unreachable, skipping reconcile until recovered")
				}
				wasHealthy = false
				continue
			}
			if !wasHealthy {
				logger.Info("temporal recovered, reconciling")
			}
			wasHealthy = true

			reconcileCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			if err := Reconcile(reconcileCtx, deps); err != nil {
				logger.Error("periodic reconciliation failed", "error", err)
			}
			cancel()
		}
	}
}
