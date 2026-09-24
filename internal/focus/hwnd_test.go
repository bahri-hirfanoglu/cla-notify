package focus

import "testing"

func TestFormatParseHWNDRoundTrip(t *testing.T) {
	got, ok := parseHWND(formatHWND(123456))
	if !ok || got != 123456 {
		t.Fatalf("round trip = %v, %v, want 123456, true", got, ok)
	}
}

func TestFormatHWNDZero(t *testing.T) {
	if got := formatHWND(0); got != "" {
		t.Fatalf("formatHWND(0) = %q, want empty", got)
	}
}

func TestParseHWNDInvalid(t *testing.T) {
	cases := []string{"", "not-a-number", "0", "-1"}
	for _, c := range cases {
		if _, ok := parseHWND(c); ok {
			t.Errorf("parseHWND(%q) accepted invalid input", c)
		}
	}
}
