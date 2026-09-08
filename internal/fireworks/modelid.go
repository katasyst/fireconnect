package fireworks

import (
	"strings"
)

var publicFireworksModelRefRE = func() func(string) (string, string, bool) {
	// Inline matcher for accounts/fireworks/models|routers/<slug>[1m]
	return func(model string) (slug, suffix string, ok bool) {
		const prefixModels = "accounts/fireworks/models/"
		const prefixRouters = "accounts/fireworks/routers/"
		lower := strings.ToLower(model)
		var rest string
		switch {
		case strings.HasPrefix(lower, prefixModels):
			rest = model[len(prefixModels):]
		case strings.HasPrefix(lower, prefixRouters):
			rest = model[len(prefixRouters):]
		default:
			return "", "", false
		}
		if strings.Contains(rest, "/") {
			return "", "", false
		}
		if strings.HasSuffix(strings.ToLower(rest), "[1m]") {
			idx := len(rest) - len("[1m]")
			return rest[:idx], "[1m]", true
		}
		return rest, "", true
	}
}()

func stripContextSuffix(model string) string {
	if strings.HasSuffix(strings.ToLower(model), "[1m]") {
		return model[:len(model)-len("[1m]")]
	}
	return model
}

// ShortFireworksModelRef converts a full Fireworks resource name to a short slug.
func ShortFireworksModelRef(model string) string {
	if slug, suffix, ok := publicFireworksModelRefRE(model); ok {
		return slug + suffix
	}
	return model
}

// NormalizeModelID normalizes a user-supplied model id to the short slug Fireworks accepts.
func NormalizeModelID(model string) string {
	bare := strings.TrimSpace(stripContextSuffix(model))
	if bare == "" {
		return bare
	}
	if IsFirerouterModel(bare) {
		return FirerouterModelID
	}
	if strings.EqualFold(bare, ClaudeNativeSlotAlias) || IsClaudeNativeModel(bare) {
		return ClaudeNativeModelID
	}
	if strings.HasPrefix(strings.ToLower(bare), "accounts/fireworks/") {
		return ShortFireworksModelRef(bare)
	}
	return bare
}

// IsClaudeNativeModel reports whether model is the reserved native-Claude slot value.
func IsClaudeNativeModel(model string) bool {
	return strings.EqualFold(stripContextSuffix(strings.TrimSpace(model)), ClaudeNativeModelID)
}

// IsFirerouterModel reports whether model refers to the firerouter router.
func IsFirerouterModel(model string) bool {
	bare := stripContextSuffix(strings.TrimSpace(model))
	parts := strings.Split(bare, "/")
	last := parts[len(parts)-1]
	return strings.EqualFold(last, FirerouterModelID)
}

// DefaultMainModelForKeyType returns the default main model for the given key type.
func DefaultMainModelForKeyType(keyType string) string {
	_ = keyType
	return DefaultMainModel
}

// WithContextTag appends [1m] to a model ID if it doesn't already have it.
// Claude Code uses this suffix to size the context window correctly.
// The gateway strips it before sending to the model.
func WithContextTag(model string) string {
	if model == "" {
		return model
	}
	if strings.HasSuffix(strings.ToLower(model), "[1m]") {
		return model
	}
	return model + "[1m]"
}

// StripContextTag removes the [1m] suffix from a model ID.
func StripContextTag(model string) string {
	if strings.HasSuffix(strings.ToLower(model), "[1m]") {
		return model[:len(model)-4]
	}
	return model
}
