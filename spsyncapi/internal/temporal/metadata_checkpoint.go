package temporal

import (
	"encoding/json"

	"spsyncapi/pkg/graphapi"
)

const (
	checkpointPhaseDrives = "drives"
	checkpointPhaseItems  = "items"
)

type driveRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type backupMetadataCheckpoint struct {
	SiteID             string                    `json:"site_id"`
	Phase              string                    `json:"phase"`
	DrivesPageURL      string                    `json:"drives_page_url,omitempty"`
	PendingDrives      []driveRef                `json:"pending_drives,omitempty"`
	CurrentDrive       *driveRef                 `json:"current_drive,omitempty"`
	CompletedDriveIDs  []string                  `json:"completed_drive_ids,omitempty"`
	Crawl              *graphapi.DriveCrawlState `json:"crawl,omitempty"`
}

type restoreMetadataCheckpoint struct {
	Marker string `json:"marker,omitempty"`
}

func parseBackupCheckpoint(raw string) backupMetadataCheckpoint {
	if raw == "" {
		return backupMetadataCheckpoint{Phase: checkpointPhaseDrives}
	}
	var cp backupMetadataCheckpoint
	_ = json.Unmarshal([]byte(raw), &cp)
	if cp.Phase == "" {
		cp.Phase = checkpointPhaseDrives
	}
	return cp
}

func encodeBackupCheckpoint(cp backupMetadataCheckpoint) string {
	b, _ := json.Marshal(cp)
	return string(b)
}

func parseRestoreCheckpoint(raw string) restoreMetadataCheckpoint {
	if raw == "" {
		return restoreMetadataCheckpoint{}
	}
	var cp restoreMetadataCheckpoint
	_ = json.Unmarshal([]byte(raw), &cp)
	return cp
}

func encodeRestoreCheckpoint(cp restoreMetadataCheckpoint) string {
	b, _ := json.Marshal(cp)
	return string(b)
}

func driveCompleted(cp *backupMetadataCheckpoint, driveID string) bool {
	for _, id := range cp.CompletedDriveIDs {
		if id == driveID {
			return true
		}
	}
	return false
}
