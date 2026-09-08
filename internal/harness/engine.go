package harness

import (
	"fmt"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/config"
	"github.com/katasyst/fireconnect/internal/ui"
)

// EngineOn runs the shared on orchestration for a harness.
func EngineOn(h Harness, ctx *cli.HarnessContext) error {
	if err := validateHarnessOn(h, ctx); err != nil {
		return err
	}
	return nil
}

// EngineOff runs the shared off orchestration for a harness.
func EngineOff(h Harness, ctx *cli.HarnessContext) error {
	enabled, err := config.IsHarnessEnabled(ctx.Home, string(h.ID()))
	if err != nil {
		return err
	}
	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), false, ""); err != nil {
		return err
	}
	if enabled {
		ui.PrintHarnessRestored(h.Label())
	} else {
		ui.Note(fmt.Sprintf("%s was not connected; nothing changed.", h.Label()))
	}
	return nil
}

func dispatchStatus(h Harness, ctx *cli.HarnessContext) error {
	enabled, err := config.IsHarnessEnabled(ctx.Home, string(h.ID()))
	if err != nil {
		return err
	}
	if ctx.JSON {
		fmt.Printf(`{"harness":%q,"enabled":%t}`+"\n", h.ID(), enabled)
		return nil
	}
	state := "off"
	if enabled {
		state = "on"
	}
	ui.Info(fmt.Sprintf("%s: %s", h.Label(), state))
	return nil
}

func validateHarnessOn(h Harness, ctx *cli.HarnessContext) error {
	if ctx.Provider != "" && !ctx.Azure {
		return fmt.Errorf("--provider is configure-only. For a one-off harness switch, use --azure")
	}
	if ctx.Azure && h.ID() == Claude {
		return fmt.Errorf("Claude does not support Azure mode; omit --azure to use the Fireworks gateway")
	}
	if ctx.Force && h.ID() != Cursor && h.ID() != Codex {
		return fmt.Errorf("--force is only supported for Cursor and Codex")
	}
	claudeFlags := ctx.Opus != "" || ctx.Sonnet != "" || ctx.Haiku != "" || ctx.Fable != "" || ctx.Subagent != ""
	if claudeFlags && h.ID() != Claude {
		return fmt.Errorf("--opus/--sonnet/--haiku/--fable/--subagent apply only to `fireconnect claude on`")
	}
	return nil
}
