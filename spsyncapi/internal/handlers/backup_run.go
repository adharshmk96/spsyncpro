package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"spsyncapi/internal/backuprun"

	"github.com/gin-gonic/gin"
)

// BackupRunHandler handles backup run read HTTP requests.
type BackupRunHandler struct {
	svc    *backuprun.Service
	logger *slog.Logger
}

// NewBackupRunHandler constructs a BackupRunHandler.
func NewBackupRunHandler(svc *backuprun.Service, logger *slog.Logger) *BackupRunHandler {
	return &BackupRunHandler{svc: svc, logger: logger}
}

type backupRunListResponse struct {
	BackupRuns []backuprun.RunDetails `json:"backup_runs"`
	Pagination Pagination             `json:"pagination"`
}

type backupRunGetResponse struct {
	BackupRun      backuprun.RunDetails            `json:"backup_run"`
	FileTransfers  []backuprun.FileTransferDetails `json:"file_transfers"`
	Pagination     Pagination                      `json:"pagination"`
}

// List returns paginated backup runs for the authenticated member.
//
// @Summary      List backup runs
// @Description  Returns paginated backup runs; optional job_id filter
// @Tags         backup-runs
// @Produce      json
// @Security     BearerAuth
// @Param        job_id  query     string  false  "Filter by backup job ID"
// @Param        page    query     int     false  "Page number (default 1)"
// @Param        limit   query     int     false  "Page size (default 20, max 100)"
// @Success      200     {object}  backupRunListResponse
// @Failure      400     {object}  errorResponse
// @Failure      401     {object}  errorResponse
// @Failure      404     {object}  errorResponse
// @Failure      500     {object}  errorResponse
// @Router       /backup-runs [get]
func (h *BackupRunHandler) List(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	page, limit, _, ok := parsePagination(c)
	if !ok {
		return
	}

	var jobID *string
	if raw := c.Query("job_id"); raw != "" {
		jobID = &raw
	}

	result, err := h.svc.List(memberID, jobID, page, limit)
	if err != nil {
		h.handleBackupRunError(c, err)
		return
	}

	runs := result.Runs
	if runs == nil {
		runs = []backuprun.RunDetails{}
	}
	c.JSON(http.StatusOK, backupRunListResponse{
		BackupRuns: runs,
		Pagination: Pagination{Page: page, Limit: limit, Total: result.Total},
	})
}

// Get returns one backup run with paginated file transfers.
//
// @Summary      Get backup run
// @Description  Returns a backup run and paginated file transfers
// @Tags         backup-runs
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      string  true   "Backup run ID"
// @Param        page   query     int     false  "File transfers page (default 1)"
// @Param        limit  query     int     false  "File transfers page size (default 20, max 100)"
// @Success      200    {object}  backupRunGetResponse
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /backup-runs/{id} [get]
func (h *BackupRunHandler) Get(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	page, limit, _, ok := parsePagination(c)
	if !ok {
		return
	}

	result, err := h.svc.Get(memberID, c.Param("id"), page, limit)
	if err != nil {
		h.handleBackupRunError(c, err)
		return
	}

	files := result.FileTransfers
	if files == nil {
		files = []backuprun.FileTransferDetails{}
	}
	c.JSON(http.StatusOK, backupRunGetResponse{
		BackupRun:     result.Run,
		FileTransfers: files,
		Pagination:    Pagination{Page: page, Limit: limit, Total: result.FilesTotal},
	})
}

// StartForJob creates and starts a backup run for a job.
//
// @Summary      Start backup run
// @Description  Creates a backup run row and starts the Temporal workflow
// @Tags         backup-runs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Backup job ID"
// @Success      201  {object}  backupRunStartResponse
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /backup-jobs/{id}/runs [post]
func (h *BackupRunHandler) StartForJob(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	details, err := h.svc.StartRun(c.Request.Context(), memberID, c.Param("id"))
	if err != nil {
		h.handleBackupRunError(c, err)
		return
	}
	c.JSON(http.StatusCreated, backupRunStartResponse{BackupRun: *details})
}

// Stop cancels an in-progress backup run.
//
// @Summary      Stop backup run
// @Description  Cancels the Temporal workflow for an in-progress backup run
// @Tags         backup-runs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Backup run ID"
// @Success      200  {object}  backupRunStartResponse
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      409  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /backup-runs/{id}/stop [post]
func (h *BackupRunHandler) Stop(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	details, err := h.svc.StopRun(c.Request.Context(), memberID, c.Param("id"))
	if err != nil {
		h.handleBackupRunError(c, err)
		return
	}
	c.JSON(http.StatusOK, backupRunStartResponse{BackupRun: *details})
}

type backupRunStartResponse struct {
	BackupRun backuprun.RunDetails `json:"backup_run"`
}

func (h *BackupRunHandler) handleBackupRunError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, backuprun.ErrBackupRunNotFound):
		respondError(c, http.StatusNotFound, "backup run not found")
	case errors.Is(err, backuprun.ErrBackupJobNotFound):
		respondError(c, http.StatusNotFound, "backup job not found")
	case errors.Is(err, backuprun.ErrRunNotInProgress):
		respondError(c, http.StatusConflict, err.Error())
	default:
		h.logger.Error("unhandled backup run error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}
