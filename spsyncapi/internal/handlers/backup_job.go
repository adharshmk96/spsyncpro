package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"spsyncapi/internal/backupjob"

	"github.com/gin-gonic/gin"
)

type BackupJobHandler struct {
	svc    *backupjob.Service
	logger *slog.Logger
}

func NewBackupJobHandler(svc *backupjob.Service, logger *slog.Logger) *BackupJobHandler {
	return &BackupJobHandler{svc: svc, logger: logger}
}

type backupJobScheduleRequest struct {
	Interval *int64     `json:"interval"`
	Cron     *string    `json:"cron"`
	OneTime  *time.Time `json:"one_time"`
}

type backupJobFiltersRequest struct {
	DocumentLibraries []string   `json:"document_libraries"`
	MinFileSize       *int64     `json:"min_file_size"`
	MaxFileSize       *int64     `json:"max_file_size"`
	CreatedAfter      *time.Time `json:"created_after"`
	UpdatedAfter      *time.Time `json:"updated_after"`
	CreatedBefore     *time.Time `json:"created_before"`
	UpdatedBefore     *time.Time `json:"updated_before"`
}

type backupJobConfigRequest struct {
	Organization string                  `json:"organization" binding:"required"`
	BucketStore  string                  `json:"bucket_store" binding:"required"`
	SharePoint   string                  `json:"share_point_site" binding:"required"`
	Filters      backupJobFiltersRequest `json:"filters"`
}

type createBackupJobRequest struct {
	LastRun   *time.Time               `json:"last_run"`
	NextRun   *time.Time               `json:"next_run"`
	StartAt   *time.Time               `json:"start_at"`
	EndAt     *time.Time               `json:"end_at"`
	Active    bool                     `json:"active"`
	Schedule  backupJobScheduleRequest `json:"schedule" binding:"required"`
	JobConfig backupJobConfigRequest   `json:"job_config" binding:"required"`
}

type updateBackupJobRequest struct {
	LastRun   *time.Time               `json:"last_run"`
	NextRun   *time.Time               `json:"next_run"`
	StartAt   *time.Time               `json:"start_at"`
	EndAt     *time.Time               `json:"end_at"`
	Active    bool                     `json:"active"`
	Schedule  backupJobScheduleRequest `json:"schedule" binding:"required"`
	JobConfig backupJobConfigRequest   `json:"job_config" binding:"required"`
}

type backupJobResponse struct {
	BackupJob backupjob.BackupJobDetails `json:"backup_job"`
}

type backupJobListResponse struct {
	BackupJobs []backupjob.BackupJobDetails `json:"backup_jobs"`
}

// Create creates a backup job.
//
// @Summary      Create backup job
// @Description  Creates a backup job with schedule and job configuration
// @Tags         backup-jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      createBackupJobRequest  true  "Backup job payload"
// @Success      201   {object}  backupJobResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /backup-jobs [post]
func (h *BackupJobHandler) Create(c *gin.Context) {
	var req createBackupJobRequest
	if !bindJSON(c, &req) {
		return
	}
	details, err := h.svc.Create(toCreateInput(req))
	if err != nil {
		h.handleBackupJobError(c, err)
		return
	}
	c.JSON(http.StatusCreated, backupJobResponse{BackupJob: *details})
}

// List returns all active backup jobs.
//
// @Summary      List backup jobs
// @Description  Returns all active backup jobs
// @Tags         backup-jobs
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  backupJobListResponse
// @Failure      401  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /backup-jobs [get]
func (h *BackupJobHandler) List(c *gin.Context) {
	jobs, err := h.svc.List()
	if err != nil {
		h.handleBackupJobError(c, err)
		return
	}
	if jobs == nil {
		jobs = []backupjob.BackupJobDetails{}
	}
	c.JSON(http.StatusOK, backupJobListResponse{BackupJobs: jobs})
}

// Get returns one active backup job by ID.
//
// @Summary      Get backup job
// @Description  Returns an active backup job by ID
// @Tags         backup-jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Backup job ID"
// @Success      200  {object}  backupJobResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /backup-jobs/{id} [get]
func (h *BackupJobHandler) Get(c *gin.Context) {
	details, err := h.svc.Get(c.Param("id"))
	if err != nil {
		h.handleBackupJobError(c, err)
		return
	}
	c.JSON(http.StatusOK, backupJobResponse{BackupJob: *details})
}

// Update modifies an active backup job.
//
// @Summary      Update backup job
// @Description  Updates a backup job and replaces schedule + job config
// @Tags         backup-jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "Backup job ID"
// @Param        body  body      updateBackupJobRequest  true  "Backup job payload"
// @Success      200   {object}  backupJobResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      404   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /backup-jobs/{id} [put]
func (h *BackupJobHandler) Update(c *gin.Context) {
	var req updateBackupJobRequest
	if !bindJSON(c, &req) {
		return
	}
	details, err := h.svc.Update(toUpdateInput(c.Param("id"), req))
	if err != nil {
		h.handleBackupJobError(c, err)
		return
	}
	c.JSON(http.StatusOK, backupJobResponse{BackupJob: *details})
}

// Delete marks a backup job inactive (soft delete).
//
// @Summary      Delete backup job
// @Description  Marks a backup job inactive; the record is retained
// @Tags         backup-jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Backup job ID"
// @Success      200  {object}  successResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /backup-jobs/{id} [delete]
func (h *BackupJobHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("id")); err != nil {
		h.handleBackupJobError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse{Success: true})
}

func (h *BackupJobHandler) handleBackupJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, backupjob.ErrInvalidSchedule),
		errors.Is(err, backupjob.ErrInvalidInterval),
		errors.Is(err, backupjob.ErrInvalidCron),
		errors.Is(err, backupjob.ErrInvalidOneTime),
		errors.Is(err, backupjob.ErrInvalidStartAt),
		errors.Is(err, backupjob.ErrInvalidOrganizationID),
		errors.Is(err, backupjob.ErrInvalidBucketStoreID),
		errors.Is(err, backupjob.ErrInvalidSharePointSite),
		errors.Is(err, backupjob.ErrInvalidMinFileSize),
		errors.Is(err, backupjob.ErrInvalidMaxFileSize),
		errors.Is(err, backupjob.ErrInvalidFileSizeRange),
		errors.Is(err, backupjob.ErrInvalidCreatedRange),
		errors.Is(err, backupjob.ErrInvalidUpdatedRange),
		errors.Is(err, backupjob.ErrInvalidDocumentLibrary):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, backupjob.ErrBackupJobNotFound):
		respondError(c, http.StatusNotFound, "backup job not found")
	default:
		h.logger.Error("unhandled backup job error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}

func toCreateInput(req createBackupJobRequest) backupjob.CreateInput {
	return backupjob.CreateInput{
		LastRun: req.LastRun,
		NextRun: req.NextRun,
		StartAt: req.StartAt,
		EndAt:   req.EndAt,
		Active:  req.Active,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: req.Schedule.Interval,
			Cron:            req.Schedule.Cron,
			OneTime:         req.Schedule.OneTime,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: req.JobConfig.Organization,
			BucketStoreID:  req.JobConfig.BucketStore,
			SharePointSite: req.JobConfig.SharePoint,
			Filters: backupjob.FilterInput{
				DocumentLibrariesCSV: strings.Join(req.JobConfig.Filters.DocumentLibraries, ","),
				MinFileSize:          req.JobConfig.Filters.MinFileSize,
				MaxFileSize:          req.JobConfig.Filters.MaxFileSize,
				CreatedAfter:         req.JobConfig.Filters.CreatedAfter,
				UpdatedAfter:         req.JobConfig.Filters.UpdatedAfter,
				CreatedBefore:        req.JobConfig.Filters.CreatedBefore,
				UpdatedBefore:        req.JobConfig.Filters.UpdatedBefore,
			},
		},
	}
}

func toUpdateInput(id string, req updateBackupJobRequest) backupjob.UpdateInput {
	in := toCreateInput(createBackupJobRequest(req))
	return backupjob.UpdateInput{
		ID:        id,
		LastRun:   in.LastRun,
		NextRun:   in.NextRun,
		StartAt:   in.StartAt,
		EndAt:     in.EndAt,
		Active:    in.Active,
		Schedule:  in.Schedule,
		JobConfig: in.JobConfig,
	}
}
