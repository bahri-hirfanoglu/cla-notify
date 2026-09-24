package present

import (
	"path/filepath"
	"testing"
)

func TestShortcutPath(t *testing.T) {
	getenv := fakeGetenv(map[string]string{"APPDATA": filepath.Join("C:", "Users", "alice", "AppData", "Roaming")})
	want := filepath.Join("C:", "Users", "alice", "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "cla-notify.lnk")
	if got := shortcutPath(getenv); got != want {
		t.Fatalf("shortcutPath() = %q, want %q", got, want)
	}
}
