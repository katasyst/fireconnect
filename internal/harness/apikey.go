package harness

import (
	"fmt"
	"os"
	"strings"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/fireworks"
	"github.com/katasyst/fireconnect/internal/keychain"
)

const missingFireworksAPIKeyMessage = "No Fireworks API key found. No settings were changed.\n\n" +
	"Sign in:\n" +
	"  fireconnect login\n\n" +
	"Custom SSO:\n" +
	"  fireconnect login --account <account-id>"

// ResolveFireworksAPIKey resolves the API key: --api-key flag > keychain > FIREWORKS_API_KEY env.
func ResolveFireworksAPIKey(ctx *cli.HarnessContext) (string, error) {
	if ctx.APIKeyFromFlag {
		key := strings.TrimSpace(ctx.APIKey)
		if key == "" {
			return "", fmt.Errorf(missingFireworksAPIKeyMessage)
		}
		if !fireworks.IsFireworksKey(key) {
			return "", fmt.Errorf("API key must start with fw_ or fpk_")
		}
		return key, nil
	}
	if key := strings.TrimSpace(ctx.APIKey); key != "" && fireworks.IsFireworksKey(key) {
		return key, nil
	}
	if secret, err := keychain.GetSecret(ctx.Home); err == nil {
		if key := strings.TrimSpace(secret); key != "" {
			return key, nil
		}
	}
	if key := strings.TrimSpace(os.Getenv("FIREWORKS_API_KEY")); key != "" {
		return key, nil
	}
	return "", fmt.Errorf(missingFireworksAPIKeyMessage)
}
