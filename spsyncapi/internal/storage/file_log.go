package storage

// Metadata sync and per-file transfer status values.
const (
	MetadataSyncNotStarted  = "not_started"
	MetadataSyncInProgress  = "in_progress"
	MetadataSyncComplete    = "complete"
	MetadataSyncFailed      = "failed"

	FileLogStatusPending = "pending"
	FileLogStatusSuccess = "success"
	FileLogStatusFailure = "failure"
)
