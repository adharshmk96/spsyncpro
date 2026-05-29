package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"spsyncapi/internal/restorejob"

	"github.com/gin-gonic/gin"
)

type RestoreJobHandler struct {
	svc    *restorejob.Service
	logger *slog.Logger
}

func NewRestoreJobHandler(svc *restorejob.Service, logger *slog.Logger) *RestoreJobHandler {
	return &RestoreJobHandler{svc: svc, logger: logger}
}

type restoreJobConfigRequest struct {
	Organization string `json:"organization" binding:"required"`
	BucketStore  string `json:"bucket_store" binding:"required"`
	SharePoint   string `json:"share_point_site" binding:"required"`
}

type createRestoreJobRequest struct {
	StartAt   *time.Time              `json:"start_at"`
	Active    *bool                   `json:"active"`
	JobConfig restoreJobConfigRequest `json:"job_config" binding:"required"`
}

type updateRestoreJobRequest struct {
	StartAt   *time.Time              `json:"start_at"`
	Active    *bool                   `json:"active"`
	JobConfig restoreJobConfigRequest `json:"job_config" binding:"required"`
}

type restoreJobResponse struct {
	RestoreJob restorejob.RestoreJobDetails `json:"restore_job"`
}

type restoreJobListResponse struct {
	RestoreJobs []restorejob.RestoreJobDetails `json:"restore_jobs"`
}

// Create creates a restore job.
//
// @Summary      Create restore job
// @Description  Creates a restore job with job configuration
// @Tags         restore-jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      createRestoreJobRequest  true  "Restore job payload"
// @Success      201   {object}  restoreJobResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /restore-jobs [post]
func (h *RestoreJobHandler) Create(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	var req createRestoreJobRequest
	if !bindJSON(c, &req) {
		return
	}
	details, err := h.svc.Create(toRestoreCreateInput(memberID, req))
	if err != nil {
		h.handleRestoreJobError(c, err)
		return
	}
	c.JSON(http.StatusCreated, restoreJobResponse{RestoreJob: *details})
}

// List returns all active restore jobs.
//
// @Summary      List restore jobs
// @Description  Returns all active restore jobs
// @Tags         restore-jobs
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  restoreJobListResponse
// @Failure      401  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /restore-jobs [get]
func (h *RestoreJobHandler) List(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	jobs, err := h.svc.List(memberID)
	if err != nil {
		h.handleRestoreJobError(c, err)
		return
	}
	if jobs == nil {
		jobs = []restorejob.RestoreJobDetails{}
	}
	c.JSON(http.StatusOK, restoreJobListResponse{RestoreJobs: jobs})
}

// Get returns one active restore job by ID.
//
// @Summary      Get restore job
// @Description  Returns an active restore job by ID
// @Tags         restore-jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Restore job ID"
// @Success      200  {object}  restoreJobResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /restore-jobs/{id} [get]
func (h *RestoreJobHandler) Get(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	details, err := h.svc.Get(memberID, c.Param("id"))
	if err != nil {
		h.handleRestoreJobError(c, err)
		return
	}
	c.JSON(http.StatusOK, restoreJobResponse{RestoreJob: *details})
}

// Update modifies an active restore job.
//
// @Summary      Update restore job
// @Description  Updates a restore job and job config
// @Tags         restore-jobs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                   true  "Restore job ID"
// @Param        body  body      updateRestoreJobRequest  true  "Restore job payload"
// @Success      200   {object}  restoreJobResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      404   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /restore-jobs/{id} [put]
func (h *RestoreJobHandler) Update(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	var req updateRestoreJobRequest
	if !bindJSON(c, &req) {
		return
	}
	details, err := h.svc.Update(memberID, toRestoreUpdateInput(c.Param("id"), req))
	if err != nil {
		h.handleRestoreJobError(c, err)
		return
	}
	c.JSON(http.StatusOK, restoreJobResponse{RestoreJob: *details})
}

// Delete marks a restore job inactive (soft delete).
//
// @Summary      Delete restore job
// @Description  Marks a restore job inactive; the record is retained
// @Tags         restore-jobs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Restore job ID"
// @Success      200  {object}  successResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /restore-jobs/{id} [delete]
func (h *RestoreJobHandler) Delete(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(memberID, c.Param("id")); err != nil {
		h.handleRestoreJobError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse{Success: true})
}

func (h *RestoreJobHandler) handleRestoreJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, restorejob.ErrInvalidStartAt),
		errors.Is(err, restorejob.ErrInvalidStartAtPast),
		errors.Is(err, restorejob.ErrInvalidOrganizationID),
		errors.Is(err, restorejob.ErrInvalidBucketStoreID),
		errors.Is(err, restorejob.ErrInvalidSharePointSite):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, restorejob.ErrRestoreJobNotFound):
		respondError(c, http.StatusNotFound, "restore job not found")
	default:
		h.logger.Error("unhandled restore job error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}

func resolveRestoreActive(active *bool) bool {
	if active == nil {
		return true
	}
	return *active
}

func toRestoreCreateInput(memberID string, req createRestoreJobRequest) restorejob.CreateInput {
	return restorejob.CreateInput{
		MemberID: memberID,
		StartAt:  req.StartAt,
		Active:   resolveRestoreActive(req.Active),
		JobConfig: restorejob.JobConfigInput{
			OrganizationID: req.JobConfig.Organization,
			BucketStoreID:  req.JobConfig.BucketStore,
			SharePointSite: req.JobConfig.SharePoint,
		},
	}
}

func toRestoreUpdateInput(id string, req updateRestoreJobRequest) restorejob.UpdateInput {
	return restorejob.UpdateInput{
		ID:      id,
		StartAt: req.StartAt,
		Active:  resolveRestoreActive(req.Active),
		JobConfig: restorejob.JobConfigInput{
			OrganizationID: req.JobConfig.Organization,
			BucketStoreID:  req.JobConfig.BucketStore,
			SharePointSite: req.JobConfig.SharePoint,
		},
	}
}
