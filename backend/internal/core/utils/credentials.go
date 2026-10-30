package utils

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,49}$`)

func ValidateUsername(name string) error {
	if !usernamePattern.MatchString(name) {
		return errors.New("use 3–50 letters, numbers, dots, underscores or hyphens, starting with a letter or number; no spaces")
	}
	return nil
}

// ValidatePassword caps passwords at 72 bytes because bcrypt ignores the rest.
func ValidatePassword(password string) error {
	if strings.TrimSpace(password) != password {
		return errors.New("password must not start or end with a space")
	}
	if utf8.RuneCountInString(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 72 {
		return errors.New("password must be at most 72 bytes")
	}
	return nil
}
