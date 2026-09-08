package harness

import (
	"fmt"
	"sync"

	"github.com/katasyst/fireconnect/internal/cli"
)

// HarnessID identifies a supported coding tool harness.
type HarnessID string

const (
	Claude HarnessID = "claude"
	Cursor HarnessID = "cursor"
	Codex  HarnessID = "codex"
)

// HarnessAliases maps user-facing aliases to canonical harness ids.
var HarnessAliases = map[string]HarnessID{
	"chatgpt": Codex,
}

// Harness patches a coding tool to route through Fireworks.
type Harness interface {
	ID() HarnessID
	Label() string
	On(ctx *cli.HarnessContext) error
	Off(ctx *cli.HarnessContext) error
	Status(ctx *cli.HarnessContext) error
}

var (
	registryMu sync.RWMutex
	registry   = map[HarnessID]Harness{}
)

// Register adds a harness to the global registry.
func Register(h Harness) {
	if h == nil {
		panic("harness: Register(nil)")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[h.ID()] = h
}

// Get returns a registered harness by id.
func Get(id HarnessID) (Harness, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	h, ok := registry[id]
	if !ok {
		return nil, fmt.Errorf("unknown harness: %s", id)
	}
	return h, nil
}

// List returns all registered harnesses in a stable order.
func List() []Harness {
	registryMu.RLock()
	defer registryMu.RUnlock()
	order := []HarnessID{Claude, Cursor, Codex}
	out := make([]Harness, 0, len(registry))
	seen := map[HarnessID]struct{}{}
	for _, id := range order {
		if h, ok := registry[id]; ok {
			out = append(out, h)
			seen[id] = struct{}{}
		}
	}
	for id, h := range registry {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, h)
	}
	return out
}

// ResolveHarnessAlias maps aliases like chatgpt to canonical harness ids.
func ResolveHarnessAlias(token string) HarnessID {
	if id, ok := HarnessAliases[token]; ok {
		return id
	}
	return HarnessID(token)
}

