package temporal

// RunKind identifies backup vs restore execution.
type RunKind string

const (
	RunKindBackup  RunKind = "backup"
	RunKindRestore RunKind = "restore"
)

// RunWorkflowInput is passed to backup/restore run workflows.
type RunWorkflowInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
	Resume   bool    `json:"resume"`
}

// ScheduledBackupInput is passed when a Temporal schedule fires a backup run.
type ScheduledBackupInput struct {
	JobID    string `json:"job_id"`
	MemberID string `json:"member_id"`
}

// FetchFileMetadataInput is the activity payload for listing files to transfer.
type FetchFileMetadataInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
}

// FetchFileMetadataOutput holds file paths discovered for a run.
type FetchFileMetadataOutput struct {
	Paths []string `json:"paths"`
}

// TransferSingleFileInput is the activity payload for one file transfer.
type TransferSingleFileInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
	FilePath string  `json:"file_path"`
}

// FinalizeRunInput is the activity payload for completing a run.
type FinalizeRunInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
}
