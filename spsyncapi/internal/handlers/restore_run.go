package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"spsyncapi/internal/restorerun"

	"github.com/gin-gonic/gin"
)

// RestoreRunHandler handles restore run read HTTP requests.
type RestoreRunHandler struct {
	svc    *restorerun.Service
	logger *slog.Logger
}

// NewRestoreRunHandler constructs a RestoreRunHandler.
func NewRestoreRunHandler(svc *restorerun.Service, logger *slog.Logger) *RestoreRunHandler {
	return &RestoreRunHandler{svc: svc, logger: logger}
}

type restoreRunListResponse struct {
	RestoreRuns []restorerun.RunDetails `json:"restore_runs"`
	Pagination  Pagination              `json:"pagination"`
}

type restoreRunGetResponse struct {
	RestoreRun     restorerun.RunDetails            `json:"restore_run"`
	FileTransfers  []restorerun.FileTransferDetails `json:"file_transfers"`
	Pagination     Pagination                       `json:"pagination"`
}

// List returns paginated restore runs for the authenticated member.
//
// @Summary      List restore runs
// @Description  Returns paginated restore runs; optional job_id filter
// @Tags         restore-runs
// @Produce      json
// @Security     BearerAuth
// @Param        job_id  query     string  false  "Filter by restore job ID"
// @Param        page    query     int     false  "Page number (default 1)"
// @Param        limit   query     int     false  "Page size (default 20, max 100)"
// @Success      200     {object}  restoreRunListResponse
// @Failure      400     {object}  errorResponse
// @Failure      401     {object}  errorResponse
// @Failure      404     {object}  errorResponse
// @Failure      500     {object}  errorResponse
// @Router       /restore-runs [get]
func (h *RestoreRunHandler) List(c *gin.Context) {
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
		h.handleRestoreRunError(c, err)
		return
	}

	runs := result.Runs
	if runs == nil {
		runs = []restorerun.RunDetails{}
	}
	c.JSON(http.StatusOK, restoreRunListResponse{
		RestoreRuns: runs,
		Pagination:  Pagination{Page: page, Limit: limit, Total: result.Total},
	})
}

// Get returns one restore run with paginated file transfers.
//
// @Summary      Get restore run
// @Description  Returns a restore run and paginated file transfers
// @Tags         restore-runs
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      string  true   "Restore run ID"
// @Param        page   query     int     false  "File transfers page (default 1)"
// @Param        limit  query     int     false  "File transfers page size (default 20, max 100)"
// @Success      200    {object}  restoreRunGetResponse
// @Failure      400    {object}  errorResponse
// @Failure      401    {object}  errorResponse
// @Failure      404    {object}  errorResponse
// @Failure      500    {object}  errorResponse
// @Router       /restore-runs/{id} [get]
func (h *RestoreRunHandler) Get(c *gin.Context) {
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
		h.handleRestoreRunError(c, err)
		return
	}

	files := result.FileTransfers
	if files == nil {
		files = []restorerun.FileTransferDetails{}
	}
	c.JSON(http.StatusOK, restoreRunGetResponse{
		RestoreRun:    result.Run,
		FileTransfers: files,
		Pagination:    Pagination{Page: page, Limit: limit, Total: result.FilesTotal},
	})
}

// StartForJob creates and starts a restore run for a job.
//
// @Summary      Start restore run
// @Description  Creates a restore run row and starts the Temporal workflow
// @Tags         restore-runs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Restore job ID"
// @Success      201  {object}  restoreRunStartResponse
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /restore-jobs/{id}/runs [post]
func (h *RestoreRunHandler) StartForJob(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	details, err := h.svc.StartRun(c.Request.Context(), memberID, c.Param("id"))
	if err != nil {
		h.handleRestoreRunError(c, err)
		return
	}
	c.JSON(http.StatusCreated, restoreRunStartResponse{RestoreRun: *details})
}

// Stop cancels an in-progress restore run.
//
// @Summary      Stop restore run
// @Description  Cancels the Temporal workflow for an in-progress restore run
// @Tags         restore-runs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Restore run ID"
// @Success      200  {object}  restoreRunStartResponse
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      409  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /restore-runs/{id}/stop [post]
func (h *RestoreRunHandler) Stop(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	details, err := h.svc.StopRun(c.Request.Context(), memberID, c.Param("id"))
	if err != nil {
		h.handleRestoreRunError(c, err)
		return
	}
	c.JSON(http.StatusOK, restoreRunStartResponse{RestoreRun: *details})
}

type restoreRunStartResponse struct {
	RestoreRun restorerun.RunDetails `json:"restore_run"`
}

func (h *RestoreRunHandler) handleRestoreRunError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, restorerun.ErrRestoreRunNotFound):
		respondError(c, http.StatusNotFound, "restore run not found")
	case errors.Is(err, restorerun.ErrRestoreJobNotFound):
		respondError(c, http.StatusNotFound, "restore job not found")
	case errors.Is(err, restorerun.ErrRunNotInProgress):
		respondError(c, http.StatusConflict, err.Error())
	default:
		h.logger.Error("unhandled restore run error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}
