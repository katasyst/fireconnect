package cli

import (
	"os"
	"path/filepath"

	"github.com/katasyst/fireconnect/internal/fireworks"
)

// HarnessContext holds all parsed CLI state for harness and global commands.
type HarnessContext struct {
	Home                 string
	SettingsPath         string
	ConfigPath           string
	DataDir              string
	DBPath               string
	APIKey               string
	APIKeyFromFlag       bool
	BaseURL              string
	BaseURLFromFlag      bool
	Azure                bool
	Provider             string
	AnthropicKey         string
	AnthropicKeyFromFlag bool
	Main                 string // --model
	Opus                 string
	Sonnet               string
	Haiku                string
	Fable                string
	Subagent             string
	JSON                 bool
	Force                bool
	Verbose              bool
	Plain                bool
	Session              string
}

// DefaultContext returns a HarnessContext with sensible defaults.
func DefaultContext() (*HarnessContext, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &HarnessContext{
		Home:    home,
		BaseURL: fireworks.FireworksBaseURL,
	}, nil
}

// DataDirPath returns the FireConnect data directory for a harness.
func (c *HarnessContext) DataDirPath(harnessID string) string {
	if c.DataDir != "" {
		return c.DataDir
	}
	return filepath.Join(c.Home, ".fireconnect", harnessID)
}
