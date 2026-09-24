package gitinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runOrSkip(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git %v failed (git not usable in this environment): %v: %s", args, err, out)
	}
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runOrSkip(t, dir, "init", "-q", "-b", "main")
	runOrSkip(t, dir, "config", "user.email", "test@example.com")
	runOrSkip(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	runOrSkip(t, dir, "add", "file.txt")
	runOrSkip(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

func TestLookupNonRepo(t *testing.T) {
	dir := t.TempDir()
	info := Lookup(dir)
	if info.Name != filepath.Base(dir) || info.Path != dir {
		t.Errorf("Lookup(%q) = %+v, want name/path only", dir, info)
	}
	if info.Branch != "" || info.Remote != "" || info.Dirty != nil {
		t.Errorf("Lookup(%q) = %+v, want no git fields outside a repo", dir, info)
	}
}

func TestLookupCleanRepo(t *testing.T) {
	dir := newRepo(t)
	info := Lookup(dir)
	if info.Branch != "main" {
		t.Errorf("Branch = %q, want main", info.Branch)
	}
	if info.Dirty == nil || *info.Dirty {
		t.Errorf("Dirty = %v, want false", info.Dirty)
	}
}

func TestLookupDirtyRepo(t *testing.T) {
	dir := newRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := Lookup(dir)
	if info.Dirty == nil || !*info.Dirty {
		t.Errorf("Dirty = %v, want true", info.Dirty)
	}
}

func TestLookupRemote(t *testing.T) {
	dir := newRepo(t)
	runOrSkip(t, dir, "remote", "add", "origin", "https://user:token@github.com/owner/repo.git")
	info := Lookup(dir)
	if info.Remote != "owner/repo" {
		t.Errorf("Remote = %q, want owner/repo", info.Remote)
	}
}

func TestNormalizeRemote(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://github.com/owner/repo.git", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"https://user:token@github.com/owner/repo.git", "owner/repo"},
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"https://gitlab.example.com/group/sub/repo.git", "gitlab.example.com/group/sub/repo"},
		{"git@gitlab.example.com:group/repo.git", "gitlab.example.com/group/repo"},
		{"", ""},
		{"not-a-url", ""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := NormalizeRemote(tt.raw); got != tt.want {
				t.Errorf("NormalizeRemote(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
