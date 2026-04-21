package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultTemporalHostPort = "localhost:7233"
	defaultTemporalNamespace = "default"
	defaultTemporalTaskQueue = "spsync-jobs"
	defaultPollIntervalSeconds = 30
)

type Config struct {
	TemporalHostPort  string
	TemporalNamespace string
	TemporalTaskQueue string
	DatabaseURL       string
	EncryptionKey     string
	PollIntervalSec   int
}

func Load() (Config, error) {
	cfg := Config{
		TemporalHostPort:  getenvDefault("TEMPORAL_HOST_PORT", defaultTemporalHostPort),
		TemporalNamespace: getenvDefault("TEMPORAL_NAMESPACE", defaultTemporalNamespace),
		TemporalTaskQueue: getenvDefault("TEMPORAL_TASK_QUEUE", defaultTemporalTaskQueue),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		EncryptionKey:     strings.TrimSpace(os.Getenv("ENCRYPTION_KEY")),
		PollIntervalSec:   defaultPollIntervalSeconds,
	}

	if rawInterval := strings.TrimSpace(os.Getenv("JOBS_RECONCILE_INTERVAL_SECONDS")); rawInterval != "" {
		parsed, err := strconv.Atoi(rawInterval)
		if err != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("invalid JOBS_RECONCILE_INTERVAL_SECONDS value: %q", rawInterval)
		}
		cfg.PollIntervalSec = parsed
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.EncryptionKey == "" {
		return Config{}, fmt.Errorf("ENCRYPTION_KEY is required")
	}

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
