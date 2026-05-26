package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"spsyncapi/internal/auth"

	"github.com/gin-gonic/gin"
)

// Context keys used to pass authenticated identity through the Gin context.
const (
	ContextKeyMemberID  = "memberId"
	ContextKeySessionID = "sessionId"
)

// Authentication returns a Gin middleware that enforces JWT-based auth.
//
// Flow:
//  1. Extract "Authorization: Bearer <token>" header.
//  2. Parse the JWT (allowing expired tokens through for step 3).
//  3a. Valid + not expired → verify session active → set context and continue.
//  3b. Valid signature but expired → verify session active → mint new JWT →
//      attach as "X-Access-Token" response header → set context and continue.
//  4. Invalid JWT or inactive session → respond 401 and abort.
func Authentication(svc *auth.Service, jwtCfg auth.JWTConfig, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := extractBearerToken(c.Request)
		if err != nil {
			abortUnauthorized(c, "missing or malformed Authorization header")
			return
		}

		result, err := auth.ParseToken(jwtCfg, tokenStr)
		if err != nil {
			logger.Debug("jwt parse failed", "error", err)
			abortUnauthorized(c, "invalid token")
			return
		}

		memberID := result.Claims.Subject
		sessionID := result.Claims.SessionID

		// Verify the referenced session is active regardless of token expiry.
		if _, err := svc.LookupSession(sessionID); err != nil {
			if errors.Is(err, auth.ErrSessionInactive) {
				abortUnauthorized(c, "session expired or revoked")
				return
			}
			logger.Error("session lookup error", "error", err)
			abortUnauthorized(c, "authentication error")
			return
		}

		// Token was expired but session is active → issue a fresh JWT.
		if result.Expired {
			newToken, err := svc.RefreshForSession(sessionID, memberID)
			if err != nil {
				logger.Error("token refresh failed", "error", err)
				abortUnauthorized(c, "authentication error")
				return
			}
			c.Header("X-Access-Token", newToken)
			logger.Debug("access token refreshed", "member_id", memberID, "session_id", sessionID)
		}

		// Inject identity into context for downstream handlers.
		c.Set(ContextKeyMemberID, memberID)
		c.Set(ContextKeySessionID, sessionID)
		c.Next()
	}
}

// GetMemberID retrieves the authenticated member ID from the Gin context.
// Returns an empty string when the context was not set by the auth middleware.
func GetMemberID(c *gin.Context) string {
	v, _ := c.Get(ContextKeyMemberID)
	id, _ := v.(string)
	return id
}

// GetSessionID retrieves the authenticated session ID from the Gin context.
func GetSessionID(c *gin.Context) string {
	v, _ := c.Get(ContextKeySessionID)
	id, _ := v.(string)
	return id
}

// extractBearerToken parses the "Authorization: Bearer <token>" header.
func extractBearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("missing Authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("Authorization header must use Bearer scheme")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}
	return token, nil
}

// abortUnauthorized writes a consistent 401 JSON response and aborts the chain.
func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})
}
