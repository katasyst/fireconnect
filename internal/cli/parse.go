package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// GlobalCommand represents a top-level fireconnect command.
type GlobalCommand struct {
	Command   string
	HelpTopic string
	Args      []string
}

// HarnessCommand represents `<harness> <verb>`.
type HarnessCommand struct {
	HarnessID string
	Verb      string
}

// ParseResult is the outcome of parsing CLI arguments.
type ParseResult struct {
	Context *HarnessContext
	Global  *GlobalCommand
	Harness *HarnessCommand
}

var globalCommands = map[string]struct{}{
	"login":  {},
	"logout": {},
	"status": {},
	"help":   {},
	"model":  {},
}

var harnessVerbs = map[string]struct{}{
	"on":     {},
	"off":    {},
	"status": {},
}

var harnessIDs = map[string]struct{}{
	"claude": {},
	"cursor": {},
	"codex":  {},
}

var harnessAliases = map[string]string{
	"chatgpt": "codex",
}

var knownFlags = []string{
	"--help", "--version", "--json", "--home", "--settings-path", "--config-path",
	"--data-dir", "--api-key", "--base-url", "--azure", "--provider",
	"--anthropic-api-key", "--model", "--opus", "--sonnet", "--haiku", "--fable",
	"--subagent", "--db-path", "--force", "--plain", "--verbose", "--session",
	"--search", "--refresh",
}

var renamedFlags = map[string]string{
	"--main":         "--model",
	"--router":       "--model firerouter",
	"--model-id":     "--model",
	"--anthropic-key": "--anthropic-api-key",
}

// ParseCLI parses argv into a global or harness command.
func ParseCLI(argv []string) (*ParseResult, error) {
	ctx, err := DefaultContext()
	if err != nil {
		return nil, err
	}

	positionals := make([]string, 0, len(argv))
	help := false
	helpTopic := ""
	version := false

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		var next string
		if i+1 < len(argv) {
			next = argv[i+1]
		}

		switch arg {
		case "--help", "-h":
			help = true
			if next != "" && !strings.HasPrefix(next, "--") {
				helpTopic = next
				i++
			} else if len(positionals) > 0 {
				helpTopic = positionals[len(positionals)-1]
			}
			continue
		case "--version", "-V":
			version = true
			continue
		}

		advance, flagErr := applyGlobalFlag(ctx, arg, next)
		if flagErr != nil {
			return nil, fmt.Errorf("%s %s", flagErr.Error(), helpHint(harnessHintToken(positionals)))
		}
		if advance > 0 {
			i += advance - 1
			continue
		}
		if strings.HasPrefix(arg, "--") {
			if replacement, ok := renamedFlags[arg]; ok {
				return nil, fmt.Errorf("unknown argument: %s. Use %s instead.", arg, replacement)
			}
			return nil, fmt.Errorf("%s %s", withSuggestion(fmt.Sprintf("unknown argument: %s.", arg), arg), helpHint(harnessHintToken(positionals)))
		}
		positionals = append(positionals, arg)
	}

	if version {
		return &ParseResult{
			Context: ctx,
			Global:  &GlobalCommand{Command: "version"},
		}, nil
	}
	if help {
		return &ParseResult{
			Context: ctx,
			Global:  &GlobalCommand{Command: "help", HelpTopic: helpTopic},
		}, nil
	}
	if len(positionals) == 0 {
		return &ParseResult{
			Context: ctx,
			Global:  &GlobalCommand{Command: "help"},
		}, nil
	}

	first := positionals[0]
	rest := positionals[1:]

	if first == "help" {
		topic := ""
		if len(rest) > 0 {
			topic = rest[0]
		}
		return &ParseResult{
			Context: ctx,
			Global:  &GlobalCommand{Command: "help", HelpTopic: topic},
		}, nil
	}

	if first == "model" {
		sub := "list"
		if len(rest) > 0 {
			sub = rest[0]
		}
		if sub != "list" {
			return nil, fmt.Errorf("unknown model subcommand: %s. Run: fireconnect model list --help", sub)
		}
		return &ParseResult{
			Context: ctx,
			Global:  &GlobalCommand{Command: "model-list", Args: rest[1:]},
		}, nil
	}

	if _, ok := globalCommands[first]; ok {
		if len(rest) > 0 {
			return nil, fmt.Errorf("%s does not accept positional arguments. %s", first, helpHint(""))
		}
		return &ParseResult{
			Context: ctx,
			Global:  &GlobalCommand{Command: first},
		}, nil
	}

	resolved := resolveHarnessToken(first)
	if _, ok := harnessIDs[string(resolved)]; ok {
		if len(rest) > 0 && rest[0] == "help" {
			return &ParseResult{
				Context: ctx,
				Global:  &GlobalCommand{Command: "help", HelpTopic: first},
			}, nil
		}
		route, err := parseHarnessRoute(resolved, rest)
		if err != nil {
			return nil, err
		}
		return &ParseResult{
			Context: ctx,
			Harness: route,
		}, nil
	}

	return nil, fmt.Errorf("%s %s", withSuggestion(fmt.Sprintf("unknown command: %s.", first), first), helpHint(""))
}

// applyGlobalFlag returns:
//
//	0 — flag not recognized
//	1 — boolean flag handled (no next-arg consumed)
//	2 — valued flag handled (next arg consumed, caller must skip it)
func applyGlobalFlag(ctx *HarnessContext, arg, next string) (int, error) {
	switch arg {
	case "--json":
		ctx.JSON = true
		return 1, nil
	case "--verbose", "-v":
		ctx.Verbose = true
		return 1, nil
	case "--plain":
		ctx.Plain = true
		return 1, nil
	case "--azure":
		ctx.Azure = true
		ctx.Provider = "azure"
		return 1, nil
	case "--force":
		ctx.Force = true
		return 1, nil
	case "--refresh":
		ctx.Refresh = true
		return 1, nil
	case "--home":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Home = value
		return 2, nil
	case "--settings-path":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.SettingsPath = value
		return 2, nil
	case "--config-path":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.ConfigPath = value
		return 2, nil
	case "--data-dir":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.DataDir = value
		return 2, nil
	case "--api-key":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.APIKey = value
		ctx.APIKeyFromFlag = true
		return 2, nil
	case "--base-url":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.BaseURL = value
		ctx.BaseURLFromFlag = true
		return 2, nil
	case "--provider":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Provider = value
		return 2, nil
	case "--anthropic-api-key":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.AnthropicKey = value
		ctx.AnthropicKeyFromFlag = true
		return 2, nil
	case "--model":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Main = value
		return 2, nil
	case "--opus":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Opus = value
		return 2, nil
	case "--sonnet":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Sonnet = value
		return 2, nil
	case "--haiku":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Haiku = value
		return 2, nil
	case "--fable":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Fable = value
		return 2, nil
	case "--subagent":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Subagent = value
		return 2, nil
	case "--db-path":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.DBPath = value
		return 2, nil
	case "--session":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Session = value
		return 2, nil
	case "--search":
		value, err := requireValue(arg, next)
		if err != nil {
			return 0, err
		}
		ctx.Search = value
		return 2, nil
	default:
		return 0, nil
	}
}

func parseHarnessRoute(id string, tokens []string) (*HarnessCommand, error) {
	if len(tokens) == 0 {
		return &HarnessCommand{HarnessID: id, Verb: "on"}, nil
	}
	verb := tokens[0]
	if _, ok := harnessVerbs[verb]; !ok {
		return nil, fmt.Errorf("%s %s", withSuggestion(fmt.Sprintf("unknown harness command: %s.", strings.Join(tokens, " ")), verb), helpHint(string(id)))
	}
	if len(tokens) > 1 {
		return nil, fmt.Errorf("fireconnect %s %s does not accept positional arguments. %s", id, verb, helpHint(string(id)))
	}
	return &HarnessCommand{HarnessID: id, Verb: verb}, nil
}

func resolveHarnessToken(token string) string {
	if id, ok := harnessAliases[token]; ok {
		return id
	}
	return token
}

func requireValue(flag, value string) (string, error) {
	if value == "" || strings.HasPrefix(value, "--") {
		return "", fmt.Errorf("%s requires a value", flag)
	}
	return value, nil
}

func helpHint(harnessID string) string {
	if harnessID != "" {
		return fmt.Sprintf("Run: fireconnect %s help", harnessID)
	}
	return "Run: fireconnect help"
}

func harnessHintToken(positionals []string) string {
	if len(positionals) == 0 {
		return ""
	}
	token := positionals[0]
	if _, ok := harnessIDs[token]; ok {
		return token
	}
	if _, ok := harnessAliases[token]; ok {
		return token
	}
	return ""
}

func withSuggestion(message, token string) string {
	if candidate := suggest(token, append(knownFlags, candidateTokens(token)...)); candidate != "" {
		return fmt.Sprintf("%s Did you mean %q?", message, candidate)
	}
	return message
}

func candidateTokens(token string) []string {
	if _, ok := harnessVerbs[token]; ok {
		return keys(harnessVerbs)
	}
	out := append([]string{}, keys(globalCommands)...)
	out = append(out, "claude", "cursor", "codex", "chatgpt")
	sort.Strings(out)
	return out
}

func suggest(token string, candidates []string) string {
	token = strings.ToLower(token)
	var best string
	bestDist := 3
	for _, c := range candidates {
		d := levenshtein(token, strings.ToLower(c))
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	if bestDist <= 2 {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				curr[j-1]+1,
				prev[j]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// ApplyHomeEnv overrides HOME when --home is not passed and HOME is unset.
func ApplyHomeEnv(ctx *HarnessContext) {
	if ctx.Home != "" {
		return
	}
	if home, err := os.UserHomeDir(); err == nil {
		ctx.Home = home
	}
}
