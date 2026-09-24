package focus

import "strconv"

// formatHWND encodes a window handle for card.FocusTarget.WindowsHWND.
func formatHWND(h uintptr) string {
	if h == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(h), 10)
}

// parseHWND decodes a card.FocusTarget.WindowsHWND value written by formatHWND.
func parseHWND(s string) (uintptr, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return uintptr(v), true
}
