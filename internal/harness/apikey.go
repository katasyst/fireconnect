package harness

import (
	"fmt"
	"os"
	"strings"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/config"
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

// ResolveAzureCredentials resolves Azure API key and base URL.
// Priority: --api-key/--base-url flags > saved config > env vars.
func ResolveAzureCredentials(ctx *cli.HarnessContext) (apiKey, baseURL string, err error) {
	if ctx.APIKeyFromFlag && ctx.APIKey != "" {
		apiKey = strings.TrimSpace(ctx.APIKey)
	}
	if ctx.BaseURLFromFlag && ctx.BaseURL != fireworks.FireworksBaseURL {
		baseURL = strings.TrimSpace(ctx.BaseURL)
	}

	if apiKey == "" || baseURL == "" {
		cfg, cfgErr := config.ReadGlobalConfig(ctx.Home)
		if cfgErr == nil {
			if apiKey == "" && cfg.Azure.APIKey != "" {
				apiKey = cfg.Azure.APIKey
			}
			if baseURL == "" && cfg.Azure.BaseURL != "" {
				baseURL = cfg.Azure.BaseURL
			}
		}
	}

	if apiKey == "" {
		if key := strings.TrimSpace(os.Getenv("AZURE_OPENAI_API_KEY")); key != "" {
			apiKey = key
		}
	}
	if baseURL == "" {
		if url := strings.TrimSpace(os.Getenv("AZURE_OPENAI_BASE_URL")); url != "" {
			baseURL = url
		}
	}

	if apiKey == "" {
		return "", "", fmt.Errorf("No Azure API key found.\n\nSet up:\n  fireconnect azure-login --api-key <key> --base-url <endpoint>")
	}
	if baseURL == "" {
		return "", "", fmt.Errorf("No Azure base URL found.\n\nSet up:\n  fireconnect azure-login --api-key <key> --base-url <endpoint>")
	}
	return apiKey, baseURL, nil
}
