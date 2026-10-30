package pos

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const defaultTransactionDescription = "Monero Merchant"
const maxTransactionDescriptionBytes = 128

func normalizeTransactionDescription(value *string) (string, error) {
	if value == nil {
		return defaultTransactionDescription, nil
	}
	if !utf8.ValidString(*value) {
		return "", errors.New("transaction description must be valid Unicode")
	}
	for _, r := range *value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return "", errors.New("transaction description must be a single line without control characters")
		}
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		normalized = defaultTransactionDescription
	}
	if len(normalized) > maxTransactionDescriptionBytes {
		return "", errors.New("transaction description must be at most 128 UTF-8 bytes")
	}
	return normalized, nil
}
