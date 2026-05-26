package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"spsyncapi/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

const instrumentationName = "spsyncapi/http"

// HTTPMetrics holds OpenTelemetry instruments for HTTP request observability.
type HTTPMetrics struct {
	RequestCount    otelmetric.Int64Counter
	RequestDuration otelmetric.Float64Histogram
	shutdown        func(context.Context) error
}

// Init configures the global OpenTelemetry meter provider and HTTP instruments.
func Init(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*HTTPMetrics, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	meterProvider, shutdown, err := newMeterProvider(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	otel.SetMeterProvider(meterProvider)
	meter := meterProvider.Meter(instrumentationName)

	requestCount, err := meter.Int64Counter(
		"http.server.request.count",
		otelmetric.WithDescription("Number of HTTP requests received"),
		otelmetric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create request count instrument: %w", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		otelmetric.WithDescription("Duration of HTTP requests"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create request duration instrument: %w", err)
	}

	return &HTTPMetrics{
		RequestCount:    requestCount,
		RequestDuration: requestDuration,
		shutdown:        shutdown,
	}, nil
}

func newMeterProvider(ctx context.Context, cfg *config.Config, logger *slog.Logger) (otelmetric.MeterProvider, func(context.Context) error, error) {
	if !cfg.Metrics.Enabled {
		logger.Info("metrics export disabled")
		return noop.NewMeterProvider(), func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.Metrics.ServiceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create otel resource: %w", err)
	}

	exporter, err := newOTLPExporter(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	reader := sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(cfg.Metrics.ExportInterval),
	)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	logger.Info("metrics export enabled",
		"service", cfg.Metrics.ServiceName,
		"otlp_endpoint", cfg.Metrics.OTLPEndpoint,
		"export_interval", cfg.Metrics.ExportInterval,
	)

	return provider, provider.Shutdown, nil
}

func newOTLPExporter(ctx context.Context, cfg *config.Config) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(normalizeOTLPEndpoint(cfg.Metrics.OTLPEndpoint)),
	}
	if cfg.Metrics.OTLPInsecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}

	return exporter, nil
}

func normalizeOTLPEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return strings.TrimSuffix(endpoint, "/")
}

// Shutdown flushes and stops the meter provider.
func (m *HTTPMetrics) Shutdown(ctx context.Context) error {
	if m == nil || m.shutdown == nil {
		return nil
	}
	return m.shutdown(ctx)
}

// RequestAttributes builds standard HTTP metric attributes for a completed request.
func RequestAttributes(method, route string, status int) otelmetric.MeasurementOption {
	return otelmetric.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", status),
	)
}

// RecordRequest records count and duration for a single HTTP request.
func (m *HTTPMetrics) RecordRequest(ctx context.Context, method, route string, status int, duration time.Duration) {
	if m == nil {
		return
	}

	opts := RequestAttributes(method, route, status)
	m.RequestCount.Add(ctx, 1, opts)
	m.RequestDuration.Record(ctx, duration.Seconds(), opts)
}
