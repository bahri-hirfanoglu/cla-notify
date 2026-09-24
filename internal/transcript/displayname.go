package transcript

import "strings"

// DisplayName turns a raw model id into a human name, e.g. "claude-opus-5-5" -> "Opus 5.5".
// A trailing 8-digit date suffix (a dated snapshot id) is stripped as noise.
func DisplayName(raw string) string {
	id := strings.TrimPrefix(raw, "claude-")
	parts := strings.Split(id, "-")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if len(last) == 8 && isAllDigits(last) {
			parts = parts[:len(parts)-1]
		}
	}
	if len(parts) == 0 || parts[0] == "" {
		return raw
	}
	family := parts[0]
	version := strings.Join(parts[1:], ".")
	displayFamily := strings.ToUpper(family[:1]) + family[1:]
	if version == "" {
		return displayFamily
	}
	return displayFamily + " " + version
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
