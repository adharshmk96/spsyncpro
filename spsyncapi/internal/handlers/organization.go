package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"spsyncapi/internal/organization"

	"github.com/gin-gonic/gin"
)

// OrganizationHandler handles organization CRUD HTTP requests.
type OrganizationHandler struct {
	svc    *organization.Service
	logger *slog.Logger
}

// NewOrganizationHandler constructs an OrganizationHandler.
func NewOrganizationHandler(svc *organization.Service, logger *slog.Logger) *OrganizationHandler {
	return &OrganizationHandler{svc: svc, logger: logger}
}

type createOrganizationRequest struct {
	Name         string `json:"name"          binding:"required"`
	TenantID     string `json:"tenant_id"     binding:"required"`
	ClientID     string `json:"client_id"     binding:"required"`
	TenantSecret string `json:"tenant_secret" binding:"required"`
}

type updateOrganizationRequest struct {
	Name         string `json:"name"          binding:"required"`
	TenantID     string `json:"tenant_id"     binding:"required"`
	ClientID     string `json:"client_id"     binding:"required"`
	TenantSecret string `json:"tenant_secret"`
}

type organizationResponse struct {
	Organization organization.OrganizationDetails `json:"organization"`
}

type organizationListResponse struct {
	Organizations []organization.OrganizationDetails `json:"organizations"`
}

// Create registers a new organization.
//
// @Summary      Create organization
// @Description  Creates an organization; tenant_secret is stored encrypted and never returned
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      createOrganizationRequest  true  "Organization payload"
// @Success      201   {object}  organizationResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      409   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /organizations [post]
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req createOrganizationRequest
	if !bindJSON(c, &req) {
		return
	}

	details, err := h.svc.Create(organization.CreateInput{
		Name:         req.Name,
		TenantID:     req.TenantID,
		ClientID:     req.ClientID,
		TenantSecret: req.TenantSecret,
	})
	if err != nil {
		h.handleOrgError(c, err)
		return
	}

	c.JSON(http.StatusCreated, organizationResponse{Organization: *details})
}

// List returns all active organizations.
//
// @Summary      List organizations
// @Description  Returns all active organizations (secrets are never included)
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  organizationListResponse
// @Failure      401  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /organizations [get]
func (h *OrganizationHandler) List(c *gin.Context) {
	orgs, err := h.svc.List()
	if err != nil {
		h.handleOrgError(c, err)
		return
	}
	if orgs == nil {
		orgs = []organization.OrganizationDetails{}
	}
	c.JSON(http.StatusOK, organizationListResponse{Organizations: orgs})
}

// Get returns one active organization by ID.
//
// @Summary      Get organization
// @Description  Returns an active organization by ID
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Organization ID"
// @Success      200  {object}  organizationResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /organizations/{id} [get]
func (h *OrganizationHandler) Get(c *gin.Context) {
	id := c.Param("id")
	details, err := h.svc.Get(id)
	if err != nil {
		h.handleOrgError(c, err)
		return
	}
	c.JSON(http.StatusOK, organizationResponse{Organization: *details})
}

// Update modifies an active organization.
//
// @Summary      Update organization
// @Description  Updates an organization; omit tenant_secret to keep the existing value
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                     true  "Organization ID"
// @Param        body  body      updateOrganizationRequest  true  "Organization payload"
// @Success      200   {object}  organizationResponse
// @Failure      400   {object}  errorResponse
// @Failure      401   {object}  errorResponse
// @Failure      404   {object}  errorResponse
// @Failure      409   {object}  errorResponse
// @Failure      500   {object}  errorResponse
// @Router       /organizations/{id} [put]
func (h *OrganizationHandler) Update(c *gin.Context) {
	var req updateOrganizationRequest
	if !bindJSON(c, &req) {
		return
	}

	details, err := h.svc.Update(organization.UpdateInput{
		ID:           c.Param("id"),
		Name:         req.Name,
		TenantID:     req.TenantID,
		ClientID:     req.ClientID,
		TenantSecret: req.TenantSecret,
	})
	if err != nil {
		h.handleOrgError(c, err)
		return
	}

	c.JSON(http.StatusOK, organizationResponse{Organization: *details})
}

// Delete marks an organization inactive (soft delete).
//
// @Summary      Delete organization
// @Description  Marks an organization inactive; the record is retained
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Organization ID"
// @Success      200  {object}  successResponse
// @Failure      401  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /organizations/{id} [delete]
func (h *OrganizationHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("id")); err != nil {
		h.handleOrgError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse{Success: true})
}

func (h *OrganizationHandler) handleOrgError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, organization.ErrInvalidName),
		errors.Is(err, organization.ErrInvalidTenantID),
		errors.Is(err, organization.ErrInvalidClientID),
		errors.Is(err, organization.ErrInvalidTenantSecret):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, organization.ErrOrganizationNotFound):
		respondError(c, http.StatusNotFound, "organization not found")
	case errors.Is(err, organization.ErrTenantIDTaken):
		respondError(c, http.StatusConflict, "tenant id already registered")
	default:
		h.logger.Error("unhandled organization error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}
