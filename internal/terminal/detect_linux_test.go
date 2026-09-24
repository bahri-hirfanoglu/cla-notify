//go:build linux

package terminal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func TestProcInfoReadsSelf(t *testing.T) {
	name, ppid, err := procInfo(os.Getpid())
	if err != nil {
		t.Fatalf("procInfo(self) error = %v", err)
	}
	if name == "" {
		t.Fatal("procInfo(self) returned an empty name")
	}
	if ppid <= 0 {
		t.Fatalf("procInfo(self) ppid = %d, want a positive pid", ppid)
	}
}

func TestProcInfoUnknownPid(t *testing.T) {
	if _, _, err := procInfo(1 << 30); err == nil {
		t.Fatal("procInfo(unknown pid) should error")
	}
}

func TestClimbToTerminalFindsMatch(t *testing.T) {
	chain := map[int][2]any{
		100: {"bash", 50},
		50:  {"tmux", 20},
		20:  {"gnome-terminal-server", 5},
		5:   {"systemd", 2},
	}
	lookup := func(pid int) (string, int, error) {
		v, ok := chain[pid]
		if !ok {
			return "", 0, os.ErrNotExist
		}
		return v[0].(string), v[1].(int), nil
	}
	if got := climbToTerminal(100, lookup); got != 20 {
		t.Fatalf("climbToTerminal() = %d, want 20 (the gnome-terminal-server ancestor)", got)
	}
}

func TestClimbToTerminalFallsBackToTopmost(t *testing.T) {
	chain := map[int][2]any{
		100: {"bash", 50},
		50:  {"sh", 1},
	}
	lookup := func(pid int) (string, int, error) {
		v, ok := chain[pid]
		if !ok {
			return "", 0, os.ErrNotExist
		}
		return v[0].(string), v[1].(int), nil
	}
	if got := climbToTerminal(100, lookup); got != 50 {
		t.Fatalf("climbToTerminal() = %d, want 50 (topmost known ancestor)", got)
	}
}

func TestTmuxClientPIDNoTmux(t *testing.T) {
	if got := tmuxClientPID(""); got != 0 {
		t.Fatalf("tmuxClientPID(empty) = %d, want 0", got)
	}
}

func TestTmuxClientPIDParsesOutput(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\necho 4321\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	t.Setenv("PATH", dir)

	if got := tmuxClientPID("/tmp/tmux-1000/default,1234,0"); got != 4321 {
		t.Fatalf("tmuxClientPID() = %d, want 4321", got)
	}
}

func TestTmuxClientPIDCommandMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if got := tmuxClientPID("/tmp/tmux-1000/default,1234,0"); got != 0 {
		t.Fatalf("tmuxClientPID() = %d, want 0 when tmux is not installed", got)
	}
}

func TestAddLinuxDetails(t *testing.T) {
	target := card.FocusTarget{}
	addLinuxDetails(&target, map[string]string{"WINDOWID": "999"})
	if target.X11WindowID != "999" {
		t.Fatalf("X11WindowID = %q, want 999", target.X11WindowID)
	}

	target = card.FocusTarget{}
	addLinuxDetails(&target, map[string]string{})
	if target.X11WindowID != "" {
		t.Fatalf("X11WindowID = %q, want empty when WINDOWID is unset", target.X11WindowID)
	}
}

func TestFindTerminalPIDSmoke(t *testing.T) {
	if pid := findTerminalPID(map[string]string{}); pid <= 0 {
		t.Fatalf("findTerminalPID() = %d, want a positive pid from the real process tree", pid)
	}
}
