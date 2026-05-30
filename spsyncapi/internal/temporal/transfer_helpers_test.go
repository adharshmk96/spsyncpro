package temporal

import (
	"testing"
	"time"

	"spsyncapi/internal/storage"
	"spsyncapi/pkg/graphapi"
)

func TestParseDocumentLibraries(t *testing.T) {
	got := parseDocumentLibraries(" Docs , Shared ")
	if len(got) != 2 || got[0] != "Docs" || got[1] != "Shared" {
		t.Fatalf("parseDocumentLibraries() = %#v", got)
	}
	if libs := parseDocumentLibraries(""); libs != nil {
		t.Fatalf("expected nil for empty csv, got %#v", libs)
	}
}

func TestBackupBlobPath(t *testing.T) {
	got := backupBlobPath("Documents", "/folder/file.txt")
	want := "Documents/folder/file.txt"
	if got != want {
		t.Fatalf("backupBlobPath() = %q, want %q", got, want)
	}
}

func TestSplitRestorePath(t *testing.T) {
	lib, path := splitRestorePath("Documents/folder/file.txt")
	if lib != "Documents" || path != "folder/file.txt" {
		t.Fatalf("splitRestorePath() = (%q, %q)", lib, path)
	}
}

func TestShouldSkipFileSizeFilters(t *testing.T) {
	min := int64(100)
	max := int64(1000)
	job := &storage.BackupJob{
		FilterMinFileSize: &min,
		FilterMaxFileSize: &max,
	}

	if !shouldSkipFile(graphapi.DriveItem{Size: 50}, job) {
		t.Fatal("expected file below min to be skipped")
	}
	if shouldSkipFile(graphapi.DriveItem{Size: 500}, job) {
		t.Fatal("expected file within range not to be skipped")
	}
	if !shouldSkipFile(graphapi.DriveItem{Size: 2000}, job) {
		t.Fatal("expected file above max to be skipped")
	}
}

func TestShouldSkipFileDateFilters(t *testing.T) {
	createdAfter := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	job := &storage.BackupJob{FilterCreatedAfter: &createdAfter}

	item := graphapi.DriveItem{
		Size:            100,
		CreatedDateTime: "2024-01-01T00:00:00Z",
	}
	if !shouldSkipFile(item, job) {
		t.Fatal("expected old created file to be skipped")
	}

	item.CreatedDateTime = "2024-07-01T00:00:00Z"
	if shouldSkipFile(item, job) {
		t.Fatal("expected recent created file not to be skipped")
	}
}

func TestDriveAllowed(t *testing.T) {
	if !driveAllowed("Docs", []string{"Docs", "Shared"}) {
		t.Fatal("expected Docs to be allowed")
	}
	if driveAllowed("Other", []string{"Docs"}) {
		t.Fatal("expected Other to be rejected")
	}
	if !driveAllowed("Anything", nil) {
		t.Fatal("expected all drives when filter empty")
	}
}

func TestEffectiveMaxConcurrentTransfers(t *testing.T) {
	if got := effectiveMaxConcurrentTransfers(0); got != defaultMaxConcurrentTransfers {
		t.Fatalf("default = %d, want %d", got, defaultMaxConcurrentTransfers)
	}
	if got := effectiveMaxConcurrentTransfers(3); got != 3 {
		t.Fatalf("explicit = %d, want 3", got)
	}
}
