package temporal

import (
	"fmt"
	"time"

	"spsyncapi/internal/storage"
)

func (a *Activities) failBackupFileTransfer(ft *storage.BackupRunFileTransfer, err error) error {
	end := time.Now().UTC()
	ft.Status = storage.FileLogStatusFailure
	ft.ErrorMessage = err.Error()
	ft.EndAt = &end
	if updateErr := a.BackupRunRepo.UpdateFileTransfer(ft); updateErr != nil {
		return fmt.Errorf("%w (also failed to persist failure: %v)", err, updateErr)
	}
	return err
}

func (a *Activities) failRestoreFileTransfer(ft *storage.RestoreRunFileTransfer, err error) error {
	end := time.Now().UTC()
	ft.Status = storage.FileLogStatusFailure
	ft.ErrorMessage = err.Error()
	ft.EndAt = &end
	if updateErr := a.RestoreRunRepo.UpdateFileTransfer(ft); updateErr != nil {
		return fmt.Errorf("%w (also failed to persist failure: %v)", err, updateErr)
	}
	return err
}
