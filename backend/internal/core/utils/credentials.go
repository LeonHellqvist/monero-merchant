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

func ValidatePassword(password string) error {
	if !utf8.ValidString(password) {
		return errors.New("password must be valid UTF-8")
	}
	if strings.ContainsAny(password, "\t\n\v\f\r \u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff") {
		return errors.New("password must not contain spaces or other whitespace")
	}
	if len(password) < 8 || len(password) > 72 {
		return errors.New("password must be 8–72 UTF-8 bytes")
	}
	return nil
}
