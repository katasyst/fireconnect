package keychain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	serviceName  = "FireworksAI"
	accountName  = "fireworks-api-key"
	apiKeyRelPath = ".fireconnect/.api-key"
)

// PlaintextSecretPath returns the fallback plaintext API key path for home.
func PlaintextSecretPath(home string) string {
	return filepath.Join(home, apiKeyRelPath)
}

// GetSecret reads the Fireworks API key from the OS keyring, falling back to the plaintext file.
func GetSecret(home string) (string, error) {
	if home = strings.TrimSpace(home); home == "" {
		return "", errors.New("HOME is not set; cannot read stored API key")
	}

	secret, err := keyring.Get(serviceName, accountName)
	if err == nil {
		if trimmed := strings.TrimSpace(secret); trimmed != "" {
			return trimmed, nil
		}
	} else if !errors.Is(err, keyring.ErrNotFound) {
		// Fall through to plaintext fallback on keyring errors.
	}

	return readPlaintextSecret(home)
}

// SetSecret stores the Fireworks API key in the OS keyring, falling back to the plaintext file.
func SetSecret(home, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("API key is required")
	}
	if home = strings.TrimSpace(home); home == "" {
		return errors.New("HOME is not set; cannot store API key")
	}

	if err := keyring.Set(serviceName, accountName, trimmed); err == nil {
		return nil
	}

	return writePlaintextSecret(home, trimmed)
}

// DeleteSecret removes the Fireworks API key from the keyring and plaintext fallback.
func DeleteSecret(home string) error {
	_ = keyring.Delete(serviceName, accountName)

	if home = strings.TrimSpace(home); home == "" {
		return nil
	}
	path := PlaintextSecretPath(home)
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// HasSecret reports whether a Fireworks API key is stored.
func HasSecret(home string) bool {
	if home = strings.TrimSpace(home); home == "" {
		return false
	}
	if secret, err := keyring.Get(serviceName, accountName); err == nil && strings.TrimSpace(secret) != "" {
		return true
	}
	secret, err := readPlaintextSecret(home)
	return err == nil && secret != ""
}

func readPlaintextSecret(home string) (string, error) {
	path := PlaintextSecretPath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", keyring.ErrNotFound
		}
		return "", err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "", keyring.ErrNotFound
	}
	return trimmed, nil
}

func writePlaintextSecret(home, value string) error {
	path := PlaintextSecretPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create secret directory: %w", err)
	}
	content := []byte(value + "\n")
	return os.WriteFile(path, content, 0o600)
}
