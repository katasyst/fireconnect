package config

import (
	"os"
	"path/filepath"

	"github.com/katasyst/fireconnect/internal/fileutil"
)

const globalConfigRelativePath = ".fireconnect/config.json"

// FireworksAPIKeyKeychainRef is the config sentinel for keychain-backed keys.
const FireworksAPIKeyKeychainRef = "{keychain:fireworks-api-key}"

// GlobalConfig holds FireConnect global settings under ~/.fireconnect/config.json.
type GlobalConfig struct {
	APIKey          string                 `json:"apiKey"`
	AnthropicAPIKey string                 `json:"anthropicApiKey"`
	SSOAccountID    string                 `json:"ssoAccountId"`
	Provider        string                 `json:"provider"`
	Azure           AzureConfig            `json:"azure"`
	Harnesses       map[string]HarnessEntry `json:"harnesses"`
}

// AzureConfig holds Azure/Foundry provider settings.
type AzureConfig struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
}

// HarnessEntry describes one harness registration in global config.
type HarnessEntry struct {
	Enabled  bool                   `json:"enabled"`
	Provider string                 `json:"provider,omitempty"`
	Profiles map[string]interface{} `json:"profiles,omitempty"`
}

var knownHarnesses = map[string]struct{}{
	"claude":   {},
	"cursor":   {},
	"codex":    {},
	"opencode": {},
	"pi":       {},
	"vscode":   {},
	"deepseek": {},
}

// GlobalConfigPath returns the path to ~/.fireconnect/config.json.
func GlobalConfigPath(home string) string {
	return filepath.Join(home, globalConfigRelativePath)
}

// ReadGlobalConfig loads global config, returning defaults when missing.
func ReadGlobalConfig(home string) (*GlobalConfig, error) {
	raw, err := fileutil.ReadJSONMap(GlobalConfigPath(home))
	if err != nil {
		return nil, err
	}
	return normalizeGlobalConfig(raw), nil
}

// WriteGlobalConfigReplace writes the full config, replacing the on-disk file.
func WriteGlobalConfigReplace(home string, config *GlobalConfig) error {
	mode := os.FileMode(0)
	if hasLiteralSecret(config) {
		mode = 0o600
	}
	return fileutil.WriteJSONFile(GlobalConfigPath(home), config, mode)
}

// WriteGlobalConfig merges patch fields into the existing config and writes the result.
func WriteGlobalConfig(home string, patch *GlobalConfig) error {
	current, err := ReadGlobalConfig(home)
	if err != nil {
		return err
	}
	if patch == nil {
		return WriteGlobalConfigReplace(home, current)
	}
	if patch.APIKey != "" {
		current.APIKey = patch.APIKey
	}
	if patch.AnthropicAPIKey != "" {
		current.AnthropicAPIKey = patch.AnthropicAPIKey
	}
	if patch.SSOAccountID != "" {
		current.SSOAccountID = patch.SSOAccountID
	}
	if patch.Provider != "" {
		current.Provider = normalizeProvider(patch.Provider)
	}
	if patch.Azure.BaseURL != "" {
		current.Azure.BaseURL = patch.Azure.BaseURL
	}
	if patch.Azure.APIKey != "" {
		current.Azure.APIKey = patch.Azure.APIKey
	}
	if patch.Harnesses != nil {
		current.Harnesses = patch.Harnesses
	}
	return WriteGlobalConfigReplace(home, current)
}

// SetHarnessEnabled updates the enabled flag for a harness.
func SetHarnessEnabled(home, harnessID string, enabled bool, provider string) error {
	cfg, err := ReadGlobalConfig(home)
	if err != nil {
		return err
	}
	if cfg.Harnesses == nil {
		cfg.Harnesses = map[string]HarnessEntry{}
	}
	entry := cfg.Harnesses[harnessID]
	entry.Enabled = enabled
	if provider == "fireworks" || provider == "azure" {
		entry.Provider = provider
	} else if !enabled {
		entry.Provider = ""
	}
	cfg.Harnesses[harnessID] = entry
	return WriteGlobalConfigReplace(home, cfg)
}

// IsHarnessEnabled reports whether a harness is marked enabled in global config.
func IsHarnessEnabled(home, harnessID string) (bool, error) {
	cfg, err := ReadGlobalConfig(home)
	if err != nil {
		return false, err
	}
	entry, ok := cfg.Harnesses[harnessID]
	if !ok {
		return false, nil
	}
	return entry.Enabled, nil
}

func normalizeGlobalConfig(raw map[string]interface{}) *GlobalConfig {
	cfg := &GlobalConfig{
		Provider:  "fireworks",
		Harnesses: map[string]HarnessEntry{},
	}
	if raw == nil {
		return cfg
	}
	if v, ok := raw["apiKey"].(string); ok {
		cfg.APIKey = v
	}
	if v, ok := raw["anthropicApiKey"].(string); ok {
		cfg.AnthropicAPIKey = v
	}
	if v, ok := raw["ssoAccountId"].(string); ok {
		cfg.SSOAccountID = v
	}
	if v, ok := raw["provider"].(string); ok {
		cfg.Provider = normalizeProvider(v)
	}
	if azureRaw, ok := raw["azure"].(map[string]interface{}); ok {
		if v, ok := azureRaw["baseUrl"].(string); ok {
			cfg.Azure.BaseURL = v
		}
		if v, ok := azureRaw["apiKey"].(string); ok {
			cfg.Azure.APIKey = v
		}
	}
	if harnessesRaw, ok := raw["harnesses"].(map[string]interface{}); ok {
		for id, entryRaw := range harnessesRaw {
			if _, known := knownHarnesses[id]; !known {
				continue
			}
			entryMap, ok := entryRaw.(map[string]interface{})
			if !ok {
				continue
			}
			entry := HarnessEntry{Enabled: entryMap["enabled"] == true}
			if provider, ok := entryMap["provider"].(string); ok {
				if provider == "azure" || provider == "fireworks" {
					entry.Provider = provider
				}
			}
			if profiles, ok := entryMap["profiles"].(map[string]interface{}); ok {
				entry.Profiles = profiles
			}
			cfg.Harnesses[id] = entry
		}
	}
	return cfg
}

func normalizeProvider(raw string) string {
	if raw == "azure" {
		return "azure"
	}
	return "fireworks"
}

func hasLiteralSecret(cfg *GlobalConfig) bool {
	if cfg.APIKey != "" && cfg.APIKey != FireworksAPIKeyKeychainRef {
		return true
	}
	if cfg.AnthropicAPIKey != "" {
		return true
	}
	if cfg.Azure.APIKey != "" {
		return true
	}
	return false
}
