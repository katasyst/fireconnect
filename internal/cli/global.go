package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/katasyst/fireconnect/internal/config"
	"github.com/katasyst/fireconnect/internal/fireworks"
	"github.com/katasyst/fireconnect/internal/keychain"
	"github.com/katasyst/fireconnect/internal/ui"
)

const version = "0.1.0-dev"

// RunGlobalCommand executes a parsed global command.
func RunGlobalCommand(cmd *GlobalCommand, ctx *HarnessContext) error {
	switch cmd.Command {
	case "version":
		fmt.Println(version)
		return nil
	case "help":
		printHelp(cmd.HelpTopic)
		return nil
	case "login":
		return runLogin(ctx)
	case "logout":
		return runLogout(ctx)
	case "status":
		return runStatus(ctx)
	case "model-list":
		return runModelList(ctx)
	case "azure-login":
		return runAzureLogin(ctx)
	default:
		return fmt.Errorf("unknown global command: %s", cmd.Command)
	}
}

func runLogin(ctx *HarnessContext) error {
	key := ctx.APIKey
	if !ctx.APIKeyFromFlag {
		return fmt.Errorf("sign in with: fireconnect login --api-key <key>")
	}
	if !fireworks.IsFireworksKey(key) {
		return fmt.Errorf("API key must start with fw_ or fpk_")
	}
	if err := keychain.SetSecret(ctx.Home, key); err != nil {
		return err
	}
	if err := config.WriteGlobalConfig(ctx.Home, &config.GlobalConfig{
		APIKey: config.FireworksAPIKeyKeychainRef,
	}); err != nil {
		return err
	}
	ui.Success("Signed in to Fireworks.")
	return nil
}

func runAzureLogin(ctx *HarnessContext) error {
	if !ctx.APIKeyFromFlag || ctx.APIKey == "" {
		return fmt.Errorf("usage: fireconnect azure-login --api-key <azure-key> --base-url <endpoint>")
	}
	if !ctx.BaseURLFromFlag || ctx.BaseURL == "" || ctx.BaseURL == fireworks.FireworksBaseURL {
		return fmt.Errorf("--base-url is required for Azure login.\nExample: fireconnect azure-login --api-key <key> --base-url https://xxx.services.ai.azure.com/openai/v1")
	}
	if err := config.WriteGlobalConfig(ctx.Home, &config.GlobalConfig{
		Azure: config.AzureConfig{
			BaseURL: ctx.BaseURL,
			APIKey:  ctx.APIKey,
		},
	}); err != nil {
		return err
	}
	ui.Success(fmt.Sprintf("Azure credentials saved. Endpoint: %s", ctx.BaseURL))
	return nil
}

func runLogout(ctx *HarnessContext) error {
	if err := keychain.DeleteSecret(ctx.Home); err != nil {
		return err
	}
	cfg, err := config.ReadGlobalConfig(ctx.Home)
	if err != nil {
		return err
	}
	cfg.APIKey = ""
	if err := config.WriteGlobalConfigReplace(ctx.Home, cfg); err != nil {
		return err
	}
	ui.Success("Signed out of Fireworks.")
	return nil
}

func runStatus(ctx *HarnessContext) error {
	hasKey := keychain.HasSecret(ctx.Home)
	cfg, err := config.ReadGlobalConfig(ctx.Home)
	if err != nil {
		return err
	}

	if ctx.JSON {
		payload := map[string]interface{}{
			"signedIn":  hasKey,
			"provider":  cfg.Provider,
			"harnesses": cfg.Harnesses,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	if hasKey {
		ui.Success("Signed in to Fireworks.")
	} else {
		ui.Warn("Fireworks: not signed in.")
		ui.Note("Run: fireconnect login --api-key <key>")
	}

	if cfg.Azure.APIKey != "" && cfg.Azure.BaseURL != "" {
		ui.Success(fmt.Sprintf("Azure configured. Endpoint: %s", cfg.Azure.BaseURL))
	}

	if !hasKey && cfg.Azure.APIKey == "" {
		return fmt.Errorf("not signed in")
	}
	return nil
}

func runModelList(ctx *HarnessContext) error {
	apiKey := ctx.APIKey
	if !ctx.APIKeyFromFlag {
		if secret, err := keychain.GetSecret(ctx.Home); err == nil {
			apiKey = secret
		}
	}
	if err := fireworks.ValidateCatalogAPIKey(apiKey); err != nil {
		return err
	}

	entries, err := fireworks.FetchCatalog(apiKey, ctx.Home, ctx.Refresh)
	if err != nil {
		return err
	}

	filtered := fireworks.FilterCatalog(entries, ctx.Search)

	if ctx.JSON {
		payload := map[string]interface{}{
			"count":  len(filtered),
			"models": filtered,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	fmt.Println(fireworks.FormatCatalogTable(filtered))
	count := len(filtered)
	ui.Note(fmt.Sprintf("%d model%s · Refresh: fireconnect model list --refresh", count, plural(count)))
	return nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func printHelp(topic string) {
	switch topic {
	case "claude", "cursor", "codex", "chatgpt":
		fmt.Printf(`Usage: fireconnect %s [command] [options]

Commands:
  on       Enable Fireworks routing (default).
  off      Restore your previous settings.
  status   Show provider, auth, and models.
  help     Show this help.

Options for on:
  --model <id>         Primary model (default: %s)
  --api-key <key>      Fireworks API key for this run
  --azure              Use Azure/Foundry provider
  --force              Write while the IDE is running (Cursor/Codex only)
  --json               Machine-readable JSON output

Claude-specific options for on:
  --opus <model>       Opus slot model
  --sonnet <model>     Sonnet slot model
  --haiku <model>      Haiku slot model
  --fable <model>      Fable slot model
  --subagent <model>   Subagent slot model

Global options:
  --home <path>        Override HOME directory
  --help               Show help
  --version            Show version
`, topic, fireworks.DefaultMainModel)
	default:
		fmt.Print(`FireConnect — use Fireworks or Azure models in AI coding tools.

Usage: fireconnect <command> [options]

Get started:
  login                Sign in to Fireworks
  azure-login          Save Azure OpenAI credentials
  claude               Route Claude Code through Fireworks

Harnesses:
  claude               Claude Code (Fireworks only)
  cursor               Cursor IDE (Fireworks only)
  codex                Codex CLI & ChatGPT app (Fireworks or Azure)
  chatgpt              Alias for codex

Per harness:
  <harness> on         Enable routing (Fireworks by default, --azure for Azure)
  <harness> off        Restore previous settings
  <harness> status     Show provider, auth, and models

Azure:
  azure-login          Save Azure credentials once
    --api-key <key>    Azure API key
    --base-url <url>   Azure endpoint (e.g. https://xxx.services.ai.azure.com/openai/v1)

  codex on --azure     Route Codex through Azure OpenAI
    --model <id>       Deployment name (required for Azure)

Other:
  model list           Browse Fireworks model catalog
  status               Sign-in state and key storage
  help                 Full command reference
  help <harness>       All options for one harness

Global options:
  --home <path>        Override HOME directory
  --api-key <key>      API key (Fireworks or Azure depending on context)
  --base-url <url>     Inference endpoint URL
  --model <id>         Primary model / deployment name
  --json               Machine-readable JSON output
  --force              Force writes while IDE is running
  --azure              Use Azure OpenAI provider (Codex only)
  --search <query>     Filter model list by name or ID
  --refresh            Force-refresh cached model catalog
  --version            Show version
`)
	}
}
