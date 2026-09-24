package present

import "path/filepath"

// registrationMarkerName records which exe path the shortcut and protocol were last
// registered for, so Show only re-registers after an upgrade moves the binary.
const registrationMarkerName = "windows-registration.json"

func registrationMarkerPath(state string) string {
	return filepath.Join(state, registrationMarkerName)
}

// registrationMarker is the JSON body of registrationMarkerPath.
type registrationMarker struct {
	ExePath string `json:"exePath"`
}
