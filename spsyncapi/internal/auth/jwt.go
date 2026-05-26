package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extends the standard JWT registered claims with our application fields.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// JWTConfig holds the parameters required to sign and verify tokens.
type JWTConfig struct {
	Secret    []byte
	Issuer    string
	AccessTTL time.Duration
}

// MintToken signs a new JWT containing the given memberID and sessionID.
func MintToken(cfg JWTConfig, memberID, sessionID string) (string, error) {
	now := time.Now()
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   memberID,
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(cfg.Secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign token: %w", err)
	}
	return signed, nil
}

// ParseResult carries the outcome of ParseToken.
type ParseResult struct {
	Claims  *Claims
	Expired bool // true when the signature is valid but the token is past its expiry
}

// ParseToken parses and validates the JWT string.
//   - If the token is fully valid, Expired is false and Claims is populated.
//   - If the token has a valid signature but is expired, Expired is true and
//     Claims is still populated so the middleware can look up the session.
//   - Any other error (bad signature, malformed, wrong issuer) returns a non-nil error.
func ParseToken(cfg JWTConfig, tokenStr string) (*ParseResult, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return cfg.Secret, nil
	}, jwt.WithIssuer(cfg.Issuer), jwt.WithExpirationRequired())

	if err == nil && token.Valid {
		return &ParseResult{Claims: claims, Expired: false}, nil
	}

	// Distinguish "token is expired" from other errors.
	// When the signature is valid but the time check fails we can still read the claims.
	if errors.Is(err, jwt.ErrTokenExpired) && claims.Subject != "" {
		return &ParseResult{Claims: claims, Expired: true}, nil
	}

	return nil, fmt.Errorf("jwt: invalid token: %w", err)
}
