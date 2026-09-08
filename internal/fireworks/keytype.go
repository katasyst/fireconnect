package fireworks

import "strings"

// IsFireworksKey reports whether key looks like a Fireworks or Fire Pass API key.
func IsFireworksKey(key string) bool {
	return strings.HasPrefix(key, "fw_") || strings.HasPrefix(key, "fpk_")
}

// DetectAPIKeyType returns "firepass" for fpk_ keys, otherwise "fireworks".
func DetectAPIKeyType(key string) string {
	trimmed := strings.TrimSpace(key)
	if strings.HasPrefix(trimmed, "fpk_") {
		return "firepass"
	}
	return "fireworks"
}

// FireworksKeyOrEmpty returns the trimmed key when shaped like Fireworks, else "".
func FireworksKeyOrEmpty(key string) string {
	if IsFireworksKey(key) {
		return strings.TrimSpace(key)
	}
	return ""
}
