package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// generateHexSecret creates a cryptographically secure random hex string.
func generateHexSecret(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(b)
}

// EnsureSecretsFile generates a secrets file at the given path if it does not exist.
// It populates auto-generated values for DB_PASSWORD, JWT_SECRET, etc.
func EnsureSecretsFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	// Determine which secrets are already present in the environment
	have := func(key string) bool { return os.Getenv(key) != "" }

	lines := []string{
		"# Auto-generated secrets for Monero Merchant",
		"# Generated on first boot. Do not delete.",
	}

	if !have("DB_PASSWORD") {
		lines = append(lines, fmt.Sprintf("DB_PASSWORD=%s", generateHexSecret(16)))
	}
	if !have("JWT_SECRET") {
		lines = append(lines, fmt.Sprintf("JWT_SECRET=%s", generateHexSecret(32)))
	}
	if !have("JWT_REFRESH_SECRET") {
		lines = append(lines, fmt.Sprintf("JWT_REFRESH_SECRET=%s", generateHexSecret(32)))
	}
	if !have("JWT_MONEROPAY_SECRET") {
		lines = append(lines, fmt.Sprintf("JWT_MONEROPAY_SECRET=%s", generateHexSecret(32)))
	}
	if !have("JWT_LWS_TOKEN") {
		lines = append(lines, fmt.Sprintf("JWT_LWS_TOKEN=%s", generateHexSecret(32)))
	}
	if !have("WALLET_PASSWORD") {
		lines = append(lines, fmt.Sprintf("WALLET_PASSWORD=%s", generateHexSecret(16)))
	}

	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}
	return nil
}

// LoadSecretsFile reads a KEY=VALUE file and sets each key as an environment variable.
func LoadSecretsFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return nil
}
