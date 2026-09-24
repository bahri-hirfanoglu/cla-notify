package present

import (
	"testing"
	"time"
)

func TestToWinRTTicksUnixEpoch(t *testing.T) {
	got := toWinRTTicks(time.Unix(0, 0).UTC())
	if got != winrtEpochOffset {
		t.Fatalf("toWinRTTicks(unix epoch) = %d, want %d", got, winrtEpochOffset)
	}
}

func TestToWinRTTicksKnownDate(t *testing.T) {
	// 2024-01-01T00:00:00Z is 1704067200 seconds after the Unix epoch.
	tm := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	want := int64(winrtEpochOffset + 1704067200*10000000)
	if got := toWinRTTicks(tm); got != want {
		t.Fatalf("toWinRTTicks(2024-01-01) = %d, want %d", got, want)
	}
}
