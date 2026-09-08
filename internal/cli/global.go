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
		return nil
	}
	ui.Warn("Not signed in.")
	ui.Note("Run: fireconnect login --api-key <key>")
	return fmt.Errorf("not signed in")
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
		fmt.Print(`FireConnect — use Fireworks models in AI coding tools.

Usage: fireconnect <command> [options]

Get started:
  login                Sign in to Fireworks
  claude               Route Claude Code through Fireworks

Harnesses:
  claude               Claude Code
  cursor               Cursor IDE
  codex                Codex CLI & ChatGPT app
  chatgpt              Alias for codex

Per harness:
  <harness> on         Enable Fireworks routing (default)
  <harness> off        Restore previous settings
  <harness> status     Show provider, auth, and models

Other:
  status               Sign-in state and key storage
  help                 Full command reference
  help <harness>       All options for one harness

Global options:
  --home <path>        Override HOME directory
  --api-key <key>      Fireworks API key
  --base-url <url>     Fireworks inference base URL
  --model <id>         Primary model
  --json               Machine-readable JSON output
  --force              Force writes while IDE is running
  --azure              Use Azure/Foundry provider
  --version            Show version
`)
	}
}
