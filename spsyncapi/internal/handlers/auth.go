package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"spsyncapi/internal/auth"
	"spsyncapi/internal/middleware"
	"spsyncapi/internal/storage"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles all authentication-related HTTP requests.
type AuthHandler struct {
	svc    *auth.Service
	logger *slog.Logger
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *auth.Service, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// --- request/response shapes -----------------------------------------------

type registerRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type resetPasswordRequest struct {
	Email    string `json:"email"    binding:"required"`
	Token    string `json:"token"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type successResponse struct {
	Success bool `json:"success"`
}

// --- handlers --------------------------------------------------------------

// Register creates a new member account.
// POST /api/v1/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.svc.Register(auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, tokenResponse{Token: result.Token})
}

// Login authenticates a member and returns a JWT.
// POST /api/v1/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.svc.Login(auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, tokenResponse{Token: result.Token})
}

// Me returns the authenticated member's profile.
// GET /api/v1/me
func (h *AuthHandler) Me(c *gin.Context) {
	memberID := middleware.GetMemberID(c)
	if memberID == "" {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	details, err := h.svc.Me(memberID)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": details})
}

// Logout revokes the current session.
// POST /api/v1/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.svc.Logout(sessionID); err != nil {
		h.logger.Error("logout failed", "error", err)
		respondError(c, http.StatusInternalServerError, "logout failed")
		return
	}

	c.JSON(http.StatusOK, successResponse{Success: true})
}

// ForgotPassword triggers a password-reset flow.
// POST /api/v1/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ForgotPassword(auth.ForgotPasswordInput{Email: req.Email}); err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResponse{Success: true})
}

// ResetPassword completes a password reset using a token.
// POST /api/v1/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.svc.ResetPassword(auth.ResetPasswordInput{
		Email:    req.Email,
		Token:    req.Token,
		Password: req.Password,
	}); err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResponse{Success: true})
}

// ChangePassword changes the password for the authenticated member.
// POST /api/v1/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if !bindJSON(c, &req) {
		return
	}

	memberID := middleware.GetMemberID(c)
	sessionID := middleware.GetSessionID(c)
	if memberID == "" {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.svc.ChangePassword(auth.ChangePasswordInput{
		MemberID:        memberID,
		SessionID:       sessionID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		h.handleAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResponse{Success: true})
}

// --- helpers ---------------------------------------------------------------

// bindJSON decodes and validates the request body, writing a 400 on failure.
// Returns false when binding failed (response already written).
func bindJSON(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return false
	}
	return true
}

// respondError writes a consistent JSON error response.
func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// handleAuthError maps domain errors to appropriate HTTP responses.
func (h *AuthHandler) handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidEmail):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrInvalidPassword):
		respondError(c, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, auth.ErrAccountNotFound):
		respondError(c, http.StatusNotFound, "account not found")
	case errors.Is(err, auth.ErrSessionInactive):
		respondError(c, http.StatusUnauthorized, "session expired or revoked")
	case errors.Is(err, storage.ErrEmailTaken):
		respondError(c, http.StatusConflict, "email already registered")
	case errors.Is(err, auth.ErrPasswordTooShort):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrPasswordTooLong):
		respondError(c, http.StatusBadRequest, err.Error())
	default:
		// Check for generic "invalid or expired" messages without wrapping sentinel errors.
		msg := err.Error()
		if strings.Contains(msg, "invalid or expired") ||
			strings.Contains(msg, "invalid credentials") {
			respondError(c, http.StatusBadRequest, msg)
			return
		}
		h.logger.Error("unhandled auth error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}
