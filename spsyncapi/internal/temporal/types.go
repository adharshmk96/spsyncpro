package temporal

// RunKind identifies backup vs restore execution.
type RunKind string

const (
	RunKindBackup  RunKind = "backup"
	RunKindRestore RunKind = "restore"
)

// FileDescriptor identifies one file to transfer between SharePoint and Azure Blob.
type FileDescriptor struct {
	Path        string `json:"path"`
	DriveID     string `json:"drive_id,omitempty"`
	DriveItemID string `json:"drive_item_id,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

// RunWorkflowInput is passed to backup/restore run workflows.
type RunWorkflowInput struct {
	RunID                  string  `json:"run_id"`
	JobID                  string  `json:"job_id"`
	MemberID               string  `json:"member_id"`
	Kind                   RunKind `json:"kind"`
	Resume                 bool    `json:"resume"`
	MaxConcurrentTransfers int     `json:"max_concurrent_transfers,omitempty"`
}

// ScheduledBackupInput is passed when a Temporal schedule fires a backup run.
type ScheduledBackupInput struct {
	JobID                  string `json:"job_id"`
	MemberID               string `json:"member_id"`
	MaxConcurrentTransfers int    `json:"max_concurrent_transfers,omitempty"`
}

// SyncFileMetadataPageInput is the activity payload for one metadata sync page.
type SyncFileMetadataPageInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
}

// SyncFileMetadataPageOutput reports metadata sync progress for one page.
type SyncFileMetadataPageOutput struct {
	Complete            bool `json:"complete"`
	FilesSyncedThisPage int  `json:"files_synced_this_page"`
}

// ListPendingFileLogsInput requests a batch of files awaiting transfer.
type ListPendingFileLogsInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
	Offset   int     `json:"offset"`
	Limit    int     `json:"limit"`
}

// ListPendingFileLogsOutput holds files ready to transfer.
type ListPendingFileLogsOutput struct {
	Files []FileDescriptor `json:"files"`
}

// TransferSingleFileInput is the activity payload for one file transfer.
type TransferSingleFileInput struct {
	RunID    string         `json:"run_id"`
	JobID    string         `json:"job_id"`
	MemberID string         `json:"member_id"`
	Kind     RunKind        `json:"kind"`
	File     FileDescriptor `json:"file"`
}

// FinalizeRunInput is the activity payload for completing a run.
type FinalizeRunInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
}
