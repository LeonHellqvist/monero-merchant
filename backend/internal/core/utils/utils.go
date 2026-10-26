package utils

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"

	"github.com/monero-merchant/monero-merchant/backend/internal/core/models"
)

func GetClaimFromContext(ctx context.Context, key models.ClaimsContextKey) (value interface{}, ok bool) {
	value = ctx.Value(key)
	if value == nil {
		return nil, false
	}
	return value, true
}

// SecureCompare compares two strings in constant time, also hiding their lengths.
func SecureCompare(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
