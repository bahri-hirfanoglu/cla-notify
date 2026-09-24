package session

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoadMissingReturnsZeroState(t *testing.T) {
	got := Load(t.TempDir(), "abc")
	if got != (State{}) {
		t.Errorf("Load() = %+v, want zero State", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	want := State{TurnStartedAt: &now, Model: "claude-opus-5-5"}
	if err := Save(dir, "session-1", want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got := Load(dir, "session-1")
	if got.Model != want.Model {
		t.Errorf("Model = %q, want %q", got.Model, want.Model)
	}
	if got.TurnStartedAt == nil || !got.TurnStartedAt.Equal(*want.TurnStartedAt) {
		t.Errorf("TurnStartedAt = %v, want %v", got.TurnStartedAt, want.TurnStartedAt)
	}
}

func TestSanitizeSessionIDPreventsPathEscape(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "../../etc/passwd", State{Model: "x"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got := Load(dir, "../../etc/passwd")
	if got.Model != "x" {
		t.Errorf("Load() = %+v, sanitized round trip should still work", got)
	}
	if sanitize("../../etc/passwd") == "../../etc/passwd" {
		t.Error("sanitize() should strip path separators")
	}
}

func TestSaveWritesPrivateFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes do not apply on windows")
	}
	dir := t.TempDir()
	if err := Save(dir, "session-1", State{Model: "x"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(sessionPath(dir, "session-1"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("session file mode = %o, want %o", got, 0o600)
	}
	dirInfo, err := os.Stat(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("sessions dir mode = %o, want %o", got, 0o700)
	}
}

func TestCleanupRemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "old", State{Model: "x"}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	path := sessionPath(dir, "old")
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if err := Save(dir, "fresh", State{Model: "y"}); err != nil {
		t.Fatal(err)
	}

	Cleanup(dir, 24*time.Hour, 24*time.Hour)

	if got := Load(dir, "old"); got.Model != "" {
		t.Errorf("Load(old) = %+v, want cleaned up (zero State)", got)
	}
	if got := Load(dir, "fresh"); got.Model != "y" {
		t.Errorf("Load(fresh) = %+v, want to survive cleanup", got)
	}
}
