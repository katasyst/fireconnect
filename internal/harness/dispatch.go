package harness

import (
	"fmt"

	"github.com/katasyst/fireconnect/internal/cli"
)

// DispatchHarnessCommand routes a harness verb to the registered harness.
func DispatchHarnessCommand(id HarnessID, verb string, ctx *cli.HarnessContext) error {
	h, err := Get(id)
	if err != nil {
		return err
	}

	switch verb {
	case "on":
		return h.On(ctx)
	case "off":
		return h.Off(ctx)
	case "status":
		return h.Status(ctx)
	default:
		return fmt.Errorf("unknown harness command: %s. Run: fireconnect %s help", verb, id)
	}
}
