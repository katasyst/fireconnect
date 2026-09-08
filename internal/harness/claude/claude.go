package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/config"
	"github.com/katasyst/fireconnect/internal/fileutil"
	"github.com/katasyst/fireconnect/internal/fireworks"
	"github.com/katasyst/fireconnect/internal/harness"
	"github.com/katasyst/fireconnect/internal/ui"
)

const (
	anthropicAPISentinel = "fireconnect"
	backupFileName       = "provider-backup.json"
	stateFileName        = "provider-state.json"
)

type claudeHarness struct{}

func init() {
	harness.Register(&claudeHarness{})
}

func (h *claudeHarness) ID() harness.HarnessID   { return harness.Claude }
func (h *claudeHarness) Label() string             { return "Claude Code" }

func (h *claudeHarness) On(ctx *cli.HarnessContext) error {
	if err := harness.EngineOn(h, ctx); err != nil {
		return err
	}

	apiKey, err := harness.ResolveFireworksAPIKey(ctx)
	if err != nil {
		return err
	}

	settingsPath := settingsPath(ctx)
	dataDir := dataDir(ctx)
	backupPath := filepath.Join(dataDir, backupFileName)
	statePath := filepath.Join(dataDir, stateFileName)

	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	if err := maybeSnapshotBackup(settingsPath, backupPath, settings); err != nil {
		return err
	}

	mapping := resolveModelMapping(ctx)
	env := settingsEnv(settings)
	for k, v := range buildFireworksEnv(apiKey, mapping, ctx) {
		env[k] = v
	}
	settings["env"] = env

	if ctx.Main != "" {
		settings["model"] = fireworks.WithContextTag(fireworks.ShortFireworksModelRef(fireworks.NormalizeModelID(ctx.Main)))
	}

	if err := fileutil.WriteJSONFile(settingsPath, settings, 0o600); err != nil {
		return err
	}

	keyType := fireworks.DetectAPIKeyType(apiKey)
	if err := fileutil.WriteJSONFile(statePath, map[string]interface{}{
		"authMode": "customHeader",
		"keyType":  keyType,
	}, 0o600); err != nil {
		return err
	}

	if err := approveStrayAnthropicAPIKey(ctx.Home); err != nil {
		return err
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), true, "fireworks"); err != nil {
		return err
	}

	model := ctx.Main
	if model == "" {
		model = fireworks.DefaultMainModel
	}
	ui.PrintHarnessConnected(h.Label(), model)
	return nil
}

func (h *claudeHarness) Off(ctx *cli.HarnessContext) error {
	settingsPath := settingsPath(ctx)
	dataDir := dataDir(ctx)
	backupPath := filepath.Join(dataDir, backupFileName)
	statePath := filepath.Join(dataDir, stateFileName)

	backup, err := fileutil.ReadJSONMap(backupPath)
	if err != nil {
		return err
	}

	if snapshot, ok := backup["snapshot"].(map[string]interface{}); ok {
		if configPath, ok := backup["configPath"].(string); ok {
			if filepath.Clean(configPath) != filepath.Clean(settingsPath) {
				return fmt.Errorf(
					"backup at %s was taken for %s, not %s; refusing to restore",
					backupPath, configPath, settingsPath,
				)
			}
		}
		existed, _ := snapshot["existed"].(bool)
		raw, _ := snapshot["raw"].(string)
		if existed {
			if err := fileutil.WriteFileAtomic(settingsPath, []byte(raw), 0); err != nil {
				return err
			}
		} else {
			if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		_ = os.Remove(backupPath)
	}

	if err := fileutil.WriteJSONFile(statePath, map[string]interface{}{}, 0o600); err != nil {
		return err
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), false, ""); err != nil {
		return err
	}

	ui.PrintHarnessRestored(h.Label())
	return nil
}

func (h *claudeHarness) Status(ctx *cli.HarnessContext) error {
	settingsPath := settingsPath(ctx)
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	env := settingsEnv(settings)
	provider := providerFromEnv(env)
	mapping := mappingFromEnv(env, settings)

	enabled, err := config.IsHarnessEnabled(ctx.Home, string(h.ID()))
	if err != nil {
		return err
	}

	if ctx.JSON {
		payload := map[string]interface{}{
			"harness":  h.ID(),
			"enabled":  enabled,
			"provider": provider,
			"baseUrl":  env["ANTHROPIC_BASE_URL"],
			"mapping":  mapping,
		}
		if model, ok := settings["model"].(string); ok && model != "" {
			payload["model"] = model
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	state := "off"
	if enabled {
		state = "on"
	}
	ui.Info(fmt.Sprintf("%s: %s · provider=%s", h.Label(), state, provider))
	if provider == "fireworks" {
		for slot, model := range mapping {
			if model != "" {
				ui.Info(fmt.Sprintf("  %s: %s", slot, fireworks.ShortFireworksModelRef(model)))
			}
		}
	}
	return nil
}

func settingsPath(ctx *cli.HarnessContext) string {
	if ctx.SettingsPath != "" {
		return ctx.SettingsPath
	}
	return filepath.Join(ctx.Home, ".claude", "settings.json")
}

func dataDir(ctx *cli.HarnessContext) string {
	if ctx.DataDir != "" {
		return ctx.DataDir
	}
	return filepath.Join(ctx.Home, ".fireconnect", "claude")
}

func claudeJSONPath(home string) string {
	return filepath.Join(home, ".claude.json")
}

func readSettings(path string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	if err := fileutil.ReadJSONFile(path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func settingsEnv(settings map[string]interface{}) map[string]string {
	raw, ok := settings["env"].(map[string]interface{})
	if !ok || raw == nil {
		return map[string]string{}
	}
	env := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			env[k] = s
		}
	}
	return env
}

func providerFromEnv(env map[string]string) string {
	if env["ANTHROPIC_BASE_URL"] == fireworks.FireworksBaseURL {
		return "fireworks"
	}
	if env["ANTHROPIC_BASE_URL"] != "" {
		return "custom"
	}
	return "default"
}

func mappingFromEnv(env map[string]string, settings map[string]interface{}) map[string]string {
	mapping := map[string]string{
		"opus":   fireworks.StripContextTag(env["ANTHROPIC_DEFAULT_OPUS_MODEL"]),
		"sonnet": fireworks.StripContextTag(env["ANTHROPIC_DEFAULT_SONNET_MODEL"]),
		"haiku":  fireworks.StripContextTag(env["ANTHROPIC_DEFAULT_HAIKU_MODEL"]),
		"fable":  fireworks.StripContextTag(env["ANTHROPIC_DEFAULT_FABLE_MODEL"]),
	}
	if v := env["CLAUDE_CODE_SUBAGENT_MODEL"]; v != "" {
		mapping["subagent"] = fireworks.StripContextTag(v)
	}
	if model, ok := settings["model"].(string); ok && model != "" {
		mapping["main"] = fireworks.StripContextTag(model)
	}
	return mapping
}

func resolveModelMapping(ctx *cli.HarnessContext) map[string]string {
	mapping := map[string]string{
		"opus":   fireworks.KimiFastLatestRouterID,
		"sonnet": fireworks.GLMFastLatestRouterID,
		"haiku":  fireworks.DeepseekFlashLatestRouterID,
		"fable":  fireworks.DeepseekProLatestRouterID,
	}
	if ctx.Opus != "" {
		mapping["opus"] = normalizeRouterRef(ctx.Opus)
	}
	if ctx.Sonnet != "" {
		mapping["sonnet"] = normalizeRouterRef(ctx.Sonnet)
	}
	if ctx.Haiku != "" {
		mapping["haiku"] = normalizeRouterRef(ctx.Haiku)
	}
	if ctx.Fable != "" {
		mapping["fable"] = normalizeRouterRef(ctx.Fable)
	}
	if ctx.Subagent != "" {
		mapping["subagent"] = normalizeRouterRef(ctx.Subagent)
	}
	return mapping
}

func normalizeRouterRef(model string) string {
	normalized := fireworks.NormalizeModelID(model)
	if strings.HasPrefix(normalized, "accounts/fireworks/") {
		return normalized
	}
	return "accounts/fireworks/routers/" + normalized
}

func buildFireworksEnv(apiKey string, mapping map[string]string, ctx *cli.HarnessContext) map[string]string {
	env := map[string]string{
		"ANTHROPIC_BASE_URL":                          fireworks.FireworksBaseURL,
		"ANTHROPIC_API_KEY":                           anthropicAPISentinel,
		"ANTHROPIC_CUSTOM_HEADERS":                    fmt.Sprintf("X-Fireworks-Api-Key: %s", apiKey),
		"ANTHROPIC_DEFAULT_OPUS_MODEL":                fireworks.WithContextTag(mapping["opus"]),
		"ANTHROPIC_DEFAULT_SONNET_MODEL":              fireworks.WithContextTag(mapping["sonnet"]),
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":               fireworks.WithContextTag(mapping["haiku"]),
		"ANTHROPIC_DEFAULT_FABLE_MODEL":               fireworks.WithContextTag(mapping["fable"]),
		"CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING":       "1",
		"CLAUDE_CODE_PACKAGE_MANAGER_AUTO_UPDATE":     "0",
		"DISABLE_TELEMETRY":                           "1",
		"DO_NOT_TRACK":                                "1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":    "1",
		"ENABLE_TOOL_SEARCH":                          "true",
	}
	if subagent, ok := mapping["subagent"]; ok && subagent != "" {
		env["CLAUDE_CODE_SUBAGENT_MODEL"] = fireworks.WithContextTag(subagent)
	}
	_ = ctx
	return env
}

func maybeSnapshotBackup(settingsPath, backupPath string, settings map[string]interface{}) error {
	existing, err := fileutil.ReadJSONMap(backupPath)
	if err != nil {
		return err
	}
	if existing["snapshot"] != nil {
		return nil
	}

	env := settingsEnv(settings)
	if providerFromEnv(env) == "fireworks" {
		return nil
	}

	snapshot, err := readRawIfExists(settingsPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o700); err != nil {
		return err
	}
	return fileutil.WriteJSONFile(backupPath, map[string]interface{}{
		"configPath": settingsPath,
		"snapshot":   snapshot,
	}, 0o600)
}

func readRawIfExists(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]interface{}{"existed": false, "raw": ""}, nil
		}
		return nil, err
	}
	return map[string]interface{}{"existed": true, "raw": string(data)}, nil
}

func approveStrayAnthropicAPIKey(home string) error {
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if home == "" || key == "" {
		return nil
	}
	identifier := key[len(key)-min(20, len(key)):]
	path := claudeJSONPath(home)

	current, err := fileutil.ReadJSONMap(path)
	if err != nil {
		return err
	}

	responses, _ := current["customApiKeyResponses"].(map[string]interface{})
	if responses == nil {
		responses = map[string]interface{}{}
	}
	approvedRaw, _ := responses["approved"].([]interface{})
	approved := make([]interface{}, 0, len(approvedRaw)+1)
	for _, item := range approvedRaw {
		if s, ok := item.(string); ok && s == identifier {
			return nil
		}
		approved = append(approved, item)
	}
	approved = append(approved, identifier)

	rejected := responses["rejected"]
	if rejected == nil {
		rejected = []interface{}{}
	}

	current["customApiKeyResponses"] = map[string]interface{}{
		"approved": approved,
		"rejected": rejected,
	}
	return fileutil.WriteJSONFile(path, current, 0o600)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
