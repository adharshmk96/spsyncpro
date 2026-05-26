package auth

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost    = bcrypt.DefaultCost
	minPassLength = 8
	maxPassLength = 72 // bcrypt silently truncates beyond 72 bytes
)

// ErrPasswordTooShort is returned when the provided password is shorter than the minimum.
var ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", minPassLength)

// ErrPasswordTooLong is returned when the provided password exceeds bcrypt's 72-byte limit.
var ErrPasswordTooLong = fmt.Errorf("password must not exceed %d characters", maxPassLength)

// HashPassword validates the plain-text password, then returns a bcrypt hash.
func HashPassword(plain string) (string, error) {
	if err := validatePasswordPolicy(plain); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword returns nil when plain matches the stored bcrypt hash, or a
// descriptive error when it does not.
func CheckPassword(plain, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return errors.New("invalid credentials")
	}
	if err != nil {
		return fmt.Errorf("check password: %w", err)
	}
	return nil
}

// validatePasswordPolicy enforces minimum/maximum length rules.
func validatePasswordPolicy(plain string) error {
	n := utf8.RuneCountInString(plain)
	if n < minPassLength {
		return ErrPasswordTooShort
	}
	if len(plain) > maxPassLength {
		return ErrPasswordTooLong
	}
	return nil
}
