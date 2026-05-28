package temporal

import "fmt"

const (
	backupSchedulePrefix = "backup-job-"
	backupWorkflowPrefix = "backup-run-"
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

func DummyFilePath(jobID string, index int) string {
	return fmt.Sprintf("/dummy/%s/file-%d.txt", jobID, index)
}
