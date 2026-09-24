package present

import (
	"path/filepath"
	"testing"
)

func fakeGetenv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestStateDirPrefersOverride(t *testing.T) {
	getenv := fakeGetenv(map[string]string{"CLA_NOTIFY_STATE_DIR": "/custom/state", "LOCALAPPDATA": "/appdata"})
	if got := stateDir(getenv); got != "/custom/state" {
		t.Fatalf("stateDir() = %q, want /custom/state", got)
	}
}

func TestStateDirFallsBackToLocalAppData(t *testing.T) {
	getenv := fakeGetenv(map[string]string{"LOCALAPPDATA": "/appdata"})
	want := filepath.Join("/appdata", "cla-notify")
	if got := stateDir(getenv); got != want {
		t.Fatalf("stateDir() = %q, want %q", got, want)
	}
}

func TestCardsDirAndCardFilePath(t *testing.T) {
	state := filepath.Join("state")
	if got, want := cardsDir(state), filepath.Join(state, "cards"); got != want {
		t.Fatalf("cardsDir() = %q, want %q", got, want)
	}
	if got, want := cardFilePath(state, "sess-1"), filepath.Join(state, "cards", "sess-1.json"); got != want {
		t.Fatalf("cardFilePath() = %q, want %q", got, want)
	}
}
