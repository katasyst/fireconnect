package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/config"
	"github.com/katasyst/fireconnect/internal/fileutil"
	"github.com/katasyst/fireconnect/internal/fireworks"
	"github.com/katasyst/fireconnect/internal/harness"
	"github.com/katasyst/fireconnect/internal/ui"
)

const (
	providerSection   = "provider.fireworks-model-catalog"
	providerHeader    = "[provider.fireworks-model-catalog]"
	envKeyRef         = "{env:FIREWORKS_API_KEY}"
	backupFileName    = "provider-backup.json"
)

type codexHarness struct{}

func init() {
	harness.Register(&codexHarness{})
}

func (h *codexHarness) ID() harness.HarnessID { return harness.Codex }
func (h *codexHarness) Label() string         { return "Codex" }

func (h *codexHarness) On(ctx *cli.HarnessContext) error {
	if err := harness.EngineOn(h, ctx); err != nil {
		return err
	}

	if harness.IsChatGPTRunning() && !ctx.Force {
		return fmt.Errorf("ChatGPT app is running. Quit ChatGPT first.")
	}

	cfgPath := configPath(ctx)
	dataDir := dataDir(ctx)
	backupPath := filepath.Join(dataDir, backupFileName)

	snapshot, err := readRawIfExists(cfgPath)
	if err != nil {
		return err
	}

	if err := maybeSnapshotBackup(cfgPath, backupPath, snapshot); err != nil {
		return err
	}

	raw := ""
	if existed, _ := snapshot["existed"].(bool); existed {
		raw, _ = snapshot["raw"].(string)
	}

	if ctx.Azure {
		return h.onAzure(ctx, cfgPath, raw)
	}
	return h.onFireworks(ctx, cfgPath, raw)
}

func (h *codexHarness) onFireworks(ctx *cli.HarnessContext, cfgPath, raw string) error {
	apiKey, err := harness.ResolveFireworksAPIKey(ctx)
	if err != nil {
		return err
	}

	model := ctx.Main
	if model == "" {
		model = fireworks.DefaultMainModel
	}
	model = fireworks.ShortFireworksModelRef(fireworks.NormalizeModelID(model))

	apiKeyValue := envKeyRef
	if ctx.APIKeyFromFlag {
		apiKeyValue = apiKey
	}

	nextRaw := patchCodexConfig(raw, model, apiKeyValue)
	if err := fileutil.WriteFileAtomic(cfgPath, []byte(nextRaw), 0o600); err != nil {
		return err
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), true, "fireworks"); err != nil {
		return err
	}

	ui.PrintHarnessConnected(h.Label(), model)
	ui.PrintRestartHint("Quit & reopen the ChatGPT app for the change to take effect.")
	return nil
}

func (h *codexHarness) onAzure(ctx *cli.HarnessContext, cfgPath, raw string) error {
	azureKey, azureURL, err := harness.ResolveAzureCredentials(ctx)
	if err != nil {
		return err
	}

	model := ctx.Main
	if model == "" {
		return fmt.Errorf("--model is required for Azure mode.\nExample: fireconnect codex on --azure --model gpt-6-luna")
	}

	nextRaw := patchCodexConfigAzure(raw, model, azureKey, azureURL)
	if err := fileutil.WriteFileAtomic(cfgPath, []byte(nextRaw), 0o600); err != nil {
		return err
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), true, "azure"); err != nil {
		return err
	}

	ui.Info(fmt.Sprintf("Codex → Azure · %s", model))
	ui.Note(fmt.Sprintf("  endpoint: %s", azureURL))
	ui.PrintRestartHint("Quit & reopen the ChatGPT app for the change to take effect.")
	return nil
}

func (h *codexHarness) Off(ctx *cli.HarnessContext) error {
	cfgPath := configPath(ctx)
	dataDir := dataDir(ctx)
	backupPath := filepath.Join(dataDir, backupFileName)

	backup, err := fileutil.ReadJSONMap(backupPath)
	if err != nil {
		return err
	}

	if snapshot, ok := backup["snapshot"].(map[string]interface{}); ok {
		if configPathVal, ok := backup["configPath"].(string); ok {
			if filepath.Clean(configPathVal) != filepath.Clean(cfgPath) {
				return fmt.Errorf(
					"backup at %s was taken for %s, not %s; refusing to restore",
					backupPath, configPathVal, cfgPath,
				)
			}
		}
		existed, _ := snapshot["existed"].(bool)
		raw, _ := snapshot["raw"].(string)
		if existed {
			if err := fileutil.WriteFileAtomic(cfgPath, []byte(raw), 0); err != nil {
				return err
			}
		} else {
			if err := os.Remove(cfgPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		_ = os.Remove(backupPath)
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), false, ""); err != nil {
		return err
	}

	ui.PrintHarnessRestored(h.Label())
	ui.PrintRestartHint("Quit & reopen the ChatGPT app for full effect.")
	return nil
}

func (h *codexHarness) Status(ctx *cli.HarnessContext) error {
	cfgPath := configPath(ctx)
	snapshot, err := readRawIfExists(cfgPath)
	if err != nil {
		return err
	}

	provider := "default"
	model := ""
	baseURL := ""

	if existed, _ := snapshot["existed"].(bool); existed {
		raw, _ := snapshot["raw"].(string)
		doc := parseTomlDoc(raw)
		if hasFireworksProvider(doc) {
			provider = "fireworks"
			if m, ok := doc["model"].(string); ok {
				model = m
			}
			if table, ok := nestedTable(doc, providerSection); ok {
				if u, ok := table["base_url"].(string); ok {
					baseURL = u
				}
			}
		} else if hasAzureProvider(doc) {
			provider = "azure"
			if m, ok := doc["model"].(string); ok {
				model = m
			}
			if table, ok := nestedTable(doc, azureProviderSection); ok {
				if u, ok := table["base_url"].(string); ok {
					baseURL = u
				}
			}
		} else if _, ok := nestedTable(doc, providerSection); ok {
			provider = "custom"
		}
	}

	enabled, err := config.IsHarnessEnabled(ctx.Home, string(h.ID()))
	if err != nil {
		return err
	}

	if ctx.JSON {
		payload := map[string]interface{}{
			"harness":  h.ID(),
			"enabled":  enabled,
			"provider": provider,
			"model":    model,
			"baseUrl":  baseURL,
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
	if model != "" {
		ui.Info(fmt.Sprintf("  model: %s", model))
	}
	return nil
}

func configPath(ctx *cli.HarnessContext) string {
	if ctx.ConfigPath != "" {
		return ctx.ConfigPath
	}
	return filepath.Join(ctx.Home, ".codex", "config.toml")
}

func dataDir(ctx *cli.HarnessContext) string {
	if ctx.DataDir != "" {
		return ctx.DataDir
	}
	return filepath.Join(ctx.Home, ".fireconnect", "codex")
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

func maybeSnapshotBackup(cfgPath, backupPath string, snapshot map[string]interface{}) error {
	existing, err := fileutil.ReadJSONMap(backupPath)
	if err != nil {
		return err
	}
	if existing["snapshot"] != nil {
		return nil
	}

	if existed, _ := snapshot["existed"].(bool); existed {
		raw, _ := snapshot["raw"].(string)
		doc := parseTomlDoc(raw)
		if hasFireworksProvider(doc) {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o700); err != nil {
		return err
	}
	return fileutil.WriteJSONFile(backupPath, map[string]interface{}{
		"configPath": cfgPath,
		"snapshot":   snapshot,
	}, 0o600)
}

func parseTomlDoc(raw string) map[string]interface{} {
	out := map[string]interface{}{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = toml.Unmarshal([]byte(raw), &out)
	return out
}

// nestedTable resolves a dotted TOML key like "provider.fireworks-model-catalog"
// through the nested map structure that BurntSushi/toml produces.
func nestedTable(doc map[string]interface{}, dottedKey string) (map[string]interface{}, bool) {
	parts := strings.SplitN(dottedKey, ".", 2)
	if len(parts) == 1 {
		table, ok := doc[parts[0]].(map[string]interface{})
		return table, ok
	}
	child, ok := doc[parts[0]].(map[string]interface{})
	if !ok {
		return nil, false
	}
	return nestedTable(child, parts[1])
}

func hasFireworksProvider(doc map[string]interface{}) bool {
	table, ok := nestedTable(doc, providerSection)
	if !ok {
		return false
	}
	baseURL, _ := table["base_url"].(string)
	return baseURL == fireworks.FireworksBaseURL
}

func hasAzureProvider(doc map[string]interface{}) bool {
	_, ok := nestedTable(doc, azureProviderSection)
	return ok
}

const (
	azureProviderSection = "provider.azure-openai"
	azureProviderHeader  = "[provider.azure-openai]"
)

var (
	rootModelLine      = regexp.MustCompile(`(?m)^model\s*=.*$`)
	rootWebSearchLine  = regexp.MustCompile(`(?m)^web_search\s*=.*$`)
	providerTableLine  = regexp.MustCompile(`(?m)^\[provider\.fireworks-model-catalog\]\s*$`)
	azureTableLine     = regexp.MustCompile(`(?m)^\[provider\.azure-openai\]\s*$`)
)

func patchCodexConfig(raw, model, apiKeyValue string) string {
	stripped := stripProviderSections(raw)
	stripped = rootModelLine.ReplaceAllString(stripped, "")
	stripped = rootWebSearchLine.ReplaceAllString(stripped, "")

	block := strings.Join([]string{
		fmt.Sprintf(`model = %q`, model),
		`web_search = "disabled"`,
		"",
		providerHeader,
		`name = "fireworks-model-catalog"`,
		fmt.Sprintf(`base_url = %q`, fireworks.FireworksBaseURL),
		fmt.Sprintf(`api_key = %q`, apiKeyValue),
		"",
	}, "\n")

	stripped = strings.TrimSpace(stripped)
	if stripped == "" {
		return block
	}
	return block + stripped + "\n"
}

func patchCodexConfigAzure(raw, model, apiKey, baseURL string) string {
	stripped := stripProviderSections(raw)
	stripped = rootModelLine.ReplaceAllString(stripped, "")
	stripped = rootWebSearchLine.ReplaceAllString(stripped, "")

	block := strings.Join([]string{
		fmt.Sprintf(`model = %q`, model),
		`web_search = "disabled"`,
		"",
		azureProviderHeader,
		`name = "azure-openai"`,
		fmt.Sprintf(`base_url = %q`, baseURL),
		fmt.Sprintf(`api_key = %q`, apiKey),
		"",
	}, "\n")

	stripped = strings.TrimSpace(stripped)
	if stripped == "" {
		return block
	}
	return block + stripped + "\n"
}

func stripProviderSections(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	skipping := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if rootModelLine.MatchString(trimmed) {
			continue
		}
		if providerTableLine.MatchString(trimmed) || azureTableLine.MatchString(trimmed) {
			skipping = true
			continue
		}
		if skipping {
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				skipping = false
				out = append(out, line)
			}
			continue
		}
		out = append(out, line)
	}

	result := strings.Join(out, "\n")
	if !strings.HasSuffix(result, "\n") && result != "" {
		result += "\n"
	}
	return result
}
