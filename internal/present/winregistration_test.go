package present

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestRegistrationMarkerPath(t *testing.T) {
	want := filepath.Join("state", "windows-registration.json")
	if got := registrationMarkerPath("state"); got != want {
		t.Fatalf("registrationMarkerPath() = %q, want %q", got, want)
	}
}

func TestRegistrationMarkerRoundTrip(t *testing.T) {
	m := registrationMarker{ExePath: `C:\bin\cla-notify.exe`}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got registrationMarker
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got != m {
		t.Fatalf("round trip = %+v, want %+v", got, m)
	}
}
