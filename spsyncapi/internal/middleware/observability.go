package middleware

import (
	"log/slog"
	"time"

	"spsyncapi/internal/telemetry"

	"github.com/gin-gonic/gin"
)

// Observability records request duration and count via OpenTelemetry and writes
// structured request logs with slog.
func Observability(logger *slog.Logger, metrics *telemetry.HTTPMetrics) gin.HandlerFunc {
	if logger == nil {
		panic("observability middleware: logger is required")
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		metrics.RecordRequest(c.Request.Context(), c.Request.Method, route, status, duration)

		attrs := []any{
			"method", c.Request.Method,
			"path", route,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"duration", duration.String(),
			"client_ip", c.ClientIP(),
		}

		const msg = "request completed"
		switch {
		case status >= 500:
			logger.Error(msg, attrs...)
		case status >= 400:
			logger.Warn(msg, attrs...)
		default:
			logger.Info(msg, attrs...)
		}
	}
}
