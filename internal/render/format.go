package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FormatTokens renders a token count compactly, e.g. 84000 -> "84k", 1000000 -> "1M".
func FormatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return trimFloat(float64(n)/1_000_000) + "M"
	case n >= 1000:
		return trimFloat(float64(n)/1000) + "k"
	default:
		return strconv.Itoa(n)
	}
}

func trimFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

// FormatContext renders the {context} placeholder, e.g. "84k / 1M".
func FormatContext(used, window int) string {
	return FormatTokens(used) + " / " + FormatTokens(window)
}

// FormatContextPercent renders the {contextPercent} placeholder, e.g. "8%".
func FormatContextPercent(used, window int) string {
	if window <= 0 {
		return "0%"
	}
	pct := int(math.Round(float64(used) / float64(window) * 100))
	return strconv.Itoa(pct) + "%"
}

// FormatDuration renders the {duration} placeholder, e.g. "2m 14s".
func FormatDuration(totalSeconds float64) string {
	total := int(math.Round(totalSeconds))
	if total < 0 {
		total = 0
	}
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
