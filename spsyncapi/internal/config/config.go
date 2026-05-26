package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Host            string
	Port            int
	GinMode         string
	logLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Metrics         MetricsConfig
}

// MetricsConfig controls OpenTelemetry metric export.
type MetricsConfig struct {
	Enabled          bool
	ServiceName      string
	OTLPEndpoint     string
	OTLPInsecure     bool
	ExportInterval   time.Duration
}

func Load() (*Config, error) {
	setDefaults()

	cfg := &Config{
		Host:            viper.GetString("server.host"),
		Port:            viper.GetInt("server.port"),
		GinMode:         viper.GetString("server.gin_mode"),
		logLevel:        viper.GetString("log.level"),
		ReadTimeout:     viper.GetDuration("server.read_timeout"),
		WriteTimeout:    viper.GetDuration("server.write_timeout"),
		ShutdownTimeout: viper.GetDuration("server.shutdown_timeout"),
		Metrics: MetricsConfig{
			Enabled:        viper.GetBool("metrics.enabled"),
			ServiceName:    viper.GetString("metrics.service_name"),
			OTLPEndpoint:   viper.GetString("metrics.otlp_endpoint"),
			OTLPInsecure:   viper.GetBool("metrics.otlp_insecure"),
			ExportInterval: viper.GetDuration("metrics.export_interval"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.gin_mode", "release")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("server.read_timeout", 15*time.Second)
	viper.SetDefault("server.write_timeout", 15*time.Second)
	viper.SetDefault("server.shutdown_timeout", 10*time.Second)
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.service_name", "spsyncapi")
	viper.SetDefault("metrics.otlp_endpoint", "localhost:4318")
	viper.SetDefault("metrics.otlp_insecure", true)
	viper.SetDefault("metrics.export_interval", 15*time.Second)
}

func (c *Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid server.port: %d", c.Port)
	}

	switch strings.ToLower(c.GinMode) {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("invalid server.gin_mode: %q", c.GinMode)
	}

	if _, err := parseLogLevel(c.logLevel); err != nil {
		return err
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("invalid server.read_timeout: %s", c.ReadTimeout)
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("invalid server.write_timeout: %s", c.WriteTimeout)
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("invalid server.shutdown_timeout: %s", c.ShutdownTimeout)
	}

	if err := c.Metrics.validate(); err != nil {
		return err
	}

	return nil
}

func (m *MetricsConfig) validate() error {
	if !m.Enabled {
		return nil
	}

	if strings.TrimSpace(m.ServiceName) == "" {
		return fmt.Errorf("invalid metrics.service_name: must not be empty when metrics are enabled")
	}

	if strings.TrimSpace(m.OTLPEndpoint) == "" {
		return fmt.Errorf("invalid metrics.otlp_endpoint: must not be empty when metrics are enabled")
	}

	if m.ExportInterval <= 0 {
		return fmt.Errorf("invalid metrics.export_interval: %s", m.ExportInterval)
	}

	return nil
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Config) SlogLevel() slog.Level {
	level, err := parseLogLevel(c.logLevel)
	if err != nil {
		return slog.LevelInfo
	}
	return level
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log.level: %q", value)
	}
}
