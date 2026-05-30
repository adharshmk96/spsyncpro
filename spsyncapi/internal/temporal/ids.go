package temporal

const (
	backupSchedulePrefix  = "backup-job-"
	backupWorkflowPrefix  = "backup-run-"
	restoreWorkflowPrefix = "restore-run-"
)

func BackupScheduleID(jobID string) string {
	return backupSchedulePrefix + jobID
}

func BackupWorkflowID(runID string) string {
	return backupWorkflowPrefix + runID
}

func RestoreWorkflowID(runID string) string {
	return restoreWorkflowPrefix + runID
}
