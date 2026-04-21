package repository

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateJobAcceptsRecurringJob(t *testing.T) {
	now := time.Now().UTC()
	recurrence := "DAILY"
	job := Job{
		ID:                  "job-1",
		RunMode:             "RECURRING",
		Recurrence:          &recurrence,
		StartAt:             &now,
		StorageConfig:       mustJSON(t, map[string]any{"azureBlobConfig": map[string]any{"containerName": "c"}}),
		FilterConfig:        mustJSON(t, map[string]any{"minFileSize": 1, "maxFileSize": 100}),
		DocumentLibraryList: []string{"docs"},
		Status:              "ACTIVE",
	}

	if err := validateJob(job); err != nil {
		t.Fatalf("expected recurring job to validate, got error: %v", err)
	}
}

func TestValidateJobRejectsInvalidRunMode(t *testing.T) {
	now := time.Now().UTC()
	job := Job{
		ID:            "job-2",
		RunMode:       "INVALID",
		StartAt:       &now,
		StorageConfig: mustJSON(t, map[string]any{}),
		FilterConfig:  mustJSON(t, map[string]any{}),
	}

	if err := validateJob(job); err == nil {
		t.Fatal("expected invalid run mode to fail validation")
	}
}

func TestValidateJobRejectsInvalidJSON(t *testing.T) {
	now := time.Now().UTC()
	recurrence := "DAILY"
	job := Job{
		ID:            "job-3",
		RunMode:       "RECURRING",
		Recurrence:    &recurrence,
		StartAt:       &now,
		StorageConfig: json.RawMessage("{"),
		FilterConfig:  mustJSON(t, map[string]any{}),
	}

	if err := validateJob(job); err == nil {
		t.Fatal("expected invalid storage JSON to fail validation")
	}
}

func mustJSON(t *testing.T, value map[string]any) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal test JSON: %v", err)
	}

	return raw
}
