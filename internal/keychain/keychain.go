package keychain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const apiKeyRelPath = ".fireconnect/.api-key"

// PlaintextSecretPath returns the API key file path.
func PlaintextSecretPath(home string) string {
	return filepath.Join(home, apiKeyRelPath)
}

// GetSecret reads the Fireworks API key from ~/.fireconnect/.api-key.
func GetSecret(home string) (string, error) {
	if home = strings.TrimSpace(home); home == "" {
		return "", errors.New("HOME is not set; cannot read stored API key")
	}
	data, err := os.ReadFile(PlaintextSecretPath(home))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("not signed in")
		}
		return "", err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "", errors.New("not signed in")
	}
	return trimmed, nil
}

// SetSecret writes the Fireworks API key to ~/.fireconnect/.api-key (mode 0600).
func SetSecret(home, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("API key is required")
	}
	if home = strings.TrimSpace(home); home == "" {
		return errors.New("HOME is not set; cannot store API key")
	}
	path := PlaintextSecretPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return os.WriteFile(path, []byte(trimmed+"\n"), 0o600)
}

// DeleteSecret removes the API key file.
func DeleteSecret(home string) error {
	if home = strings.TrimSpace(home); home == "" {
		return nil
	}
	err := os.Remove(PlaintextSecretPath(home))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// HasSecret reports whether an API key file exists and is non-empty.
func HasSecret(home string) bool {
	secret, err := GetSecret(home)
	return err == nil && secret != ""
}
