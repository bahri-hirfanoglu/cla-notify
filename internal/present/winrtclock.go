package present

import "time"

// winrtEpochOffset is the number of 100ns ticks between the WinRT/FILETIME epoch
// (1601-01-01 UTC) and the Unix epoch (1970-01-01 UTC).
const winrtEpochOffset = 116444736000000000

// toWinRTTicks converts t to a Windows.Foundation.DateTime UniversalTime value:
// 100ns ticks since 1601-01-01 UTC.
func toWinRTTicks(t time.Time) int64 {
	return t.UTC().UnixNano()/100 + winrtEpochOffset
}
