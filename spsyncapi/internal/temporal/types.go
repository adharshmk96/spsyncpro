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

// TransferFilesInput is the activity payload for dummy file transfers.
type TransferFilesInput struct {
	RunID    string  `json:"run_id"`
	JobID    string  `json:"job_id"`
	MemberID string  `json:"member_id"`
	Kind     RunKind `json:"kind"`
}
