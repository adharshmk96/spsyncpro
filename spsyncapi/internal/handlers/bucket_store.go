package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"spsyncapi/internal/bucketstore"

	"github.com/gin-gonic/gin"
)

// BucketStoreHandler handles bucket store CRUD HTTP requests.
type BucketStoreHandler struct {
	svc    *bucketstore.Service
	logger *slog.Logger
}

// NewBucketStoreHandler constructs a BucketStoreHandler.
func NewBucketStoreHandler(svc *bucketstore.Service, logger *slog.Logger) *BucketStoreHandler {
	return &BucketStoreHandler{svc: svc, logger: logger}
}

type createBucketStoreRequest struct {
	BucketName string          `json:"bucket_name" binding:"required"`
	BucketType string          `json:"bucket_type" binding:"required"`
	Config     json.RawMessage `json:"config"      binding:"required"`
}

type updateBucketStoreRequest struct {
	BucketName string          `json:"bucket_name" binding:"required"`
	BucketType string          `json:"bucket_type" binding:"required"`
	Config     json.RawMessage `json:"config"`
}

type bucketStoreResponse struct {
	BucketStore bucketstore.BucketStoreDetails `json:"bucket_store"`
}

type bucketStoreListResponse struct {
	BucketStores []bucketstore.BucketStoreDetails `json:"bucket_stores"`
}

// Create registers a new bucket store.
//
// @Summary      Create bucket store
// @Description  Creates a bucket store; config is stored encrypted and never returned
// @Tags         bucket-stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      createBucketStoreRequest  true  "Bucket store payload"
// @Success      201   {object}  bucketStoreResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      409   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /bucket-stores [post]
func (h *BucketStoreHandler) Create(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	var req createBucketStoreRequest
	if !bindJSON(c, &req) {
		return
	}

	details, err := h.svc.Create(bucketstore.CreateInput{
		MemberID:   memberID,
		BucketName: req.BucketName,
		BucketType: req.BucketType,
		Config:     req.Config,
	})
	if err != nil {
		h.handleBucketStoreError(c, err)
		return
	}

	c.JSON(http.StatusCreated, bucketStoreResponse{BucketStore: *details})
}

// List returns all active bucket stores.
//
// @Summary      List bucket stores
// @Description  Returns all active bucket stores (config is never included)
// @Tags         bucket-stores
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  bucketStoreListResponse
// @Failure      401  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /bucket-stores [get]
func (h *BucketStoreHandler) List(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	stores, err := h.svc.List(memberID)
	if err != nil {
		h.handleBucketStoreError(c, err)
		return
	}
	if stores == nil {
		stores = []bucketstore.BucketStoreDetails{}
	}
	c.JSON(http.StatusOK, bucketStoreListResponse{BucketStores: stores})
}

// Get returns one active bucket store by ID.
//
// @Summary      Get bucket store
// @Description  Returns an active bucket store by ID
// @Tags         bucket-stores
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Bucket store ID"
// @Success      200  {object}  bucketStoreResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /bucket-stores/{id} [get]
func (h *BucketStoreHandler) Get(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	details, err := h.svc.Get(memberID, id)
	if err != nil {
		h.handleBucketStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, bucketStoreResponse{BucketStore: *details})
}

// Update modifies an active bucket store.
//
// @Summary      Update bucket store
// @Description  Updates a bucket store; omit config to keep the existing value
// @Tags         bucket-stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "Bucket store ID"
// @Param        body  body      updateBucketStoreRequest  true  "Bucket store payload"
// @Success      200   {object}  bucketStoreResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      404   {object}  errorResponse
// @Failure      409   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /bucket-stores/{id} [put]
func (h *BucketStoreHandler) Update(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	var req updateBucketStoreRequest
	if !bindJSON(c, &req) {
		return
	}

	details, err := h.svc.Update(memberID, bucketstore.UpdateInput{
		ID:         c.Param("id"),
		BucketName: req.BucketName,
		BucketType: req.BucketType,
		Config:     req.Config,
	})
	if err != nil {
		h.handleBucketStoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, bucketStoreResponse{BucketStore: *details})
}

// Delete marks a bucket store inactive (soft delete).
//
// @Summary      Delete bucket store
// @Description  Marks a bucket store inactive; the record is retained
// @Tags         bucket-stores
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Bucket store ID"
// @Success      200  {object}  successResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /bucket-stores/{id} [delete]
func (h *BucketStoreHandler) Delete(c *gin.Context) {
	memberID, ok := requireMemberID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(memberID, c.Param("id")); err != nil {
		h.handleBucketStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse{Success: true})
}

func (h *BucketStoreHandler) handleBucketStoreError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, bucketstore.ErrInvalidBucketName),
		errors.Is(err, bucketstore.ErrInvalidBucketType),
		errors.Is(err, bucketstore.ErrInvalidConfig):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, bucketstore.ErrBucketStoreNotFound):
		respondError(c, http.StatusNotFound, "bucket store not found")
	case errors.Is(err, bucketstore.ErrBucketNameTaken):
		respondError(c, http.StatusConflict, "bucket name already registered")
	default:
		h.logger.Error("unhandled bucket store error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}
