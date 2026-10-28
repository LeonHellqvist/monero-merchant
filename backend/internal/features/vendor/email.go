package vendor

import (
	"errors"
	"net/mail"
	"strings"
)

const maxEmailLength = 254

// normalizeEmail accepts a single bare mailbox, preserving the local part and
// lowercasing only its domain. Missing optional emails are stored as SQL NULL.
func normalizeEmail(input string) (*string, error) {
	email := strings.TrimSpace(input)
	if email == "" {
		return nil, nil
	}
	if len(email) > maxEmailLength {
		return nil, errors.New("email must be no more than 254 bytes")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Name != "" || address.Address != email {
		return nil, errors.New("invalid email address")
	}
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at > 64 {
		return nil, errors.New("invalid email address")
	}
	domain := email[at+1:]
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return nil, errors.New("invalid email address")
		}
		for _, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return nil, errors.New("invalid email address")
			}
		}
	}
	email = email[:at+1] + strings.ToLower(domain)
	return &email, nil
}
