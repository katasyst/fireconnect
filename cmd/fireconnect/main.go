package main

import (
	"os"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/harness"
	_ "github.com/katasyst/fireconnect/internal/harness/claude"
	_ "github.com/katasyst/fireconnect/internal/harness/codex"
	_ "github.com/katasyst/fireconnect/internal/harness/cursor"
	"github.com/katasyst/fireconnect/internal/ui"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	parsed, err := cli.ParseCLI(argv)
	if err != nil {
		ui.Error(err.Error())
		return 1
	}

	cli.ApplyHomeEnv(parsed.Context)

	if parsed.Global != nil {
		if err := cli.RunGlobalCommand(parsed.Global, parsed.Context); err != nil {
			ui.Error(err.Error())
			return 1
		}
		return 0
	}

	if parsed.Harness != nil {
		if err := harness.DispatchHarnessCommand(harness.HarnessID(parsed.Harness.HarnessID), parsed.Harness.Verb, parsed.Context); err != nil {
			ui.Error(err.Error())
			return 1
		}
		return 0
	}

	ui.Error("internal error: empty parse result")
	return 1
}
