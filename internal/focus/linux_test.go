//go:build linux

package focus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

// writeFakeTool drops an executable shell script on dir/name that appends its args to logPath.
func writeFakeTool(t *testing.T, dir, name, logPath string) {
	t.Helper()
	script := "#!/bin/sh\necho \"$0 $*\" >> \"" + logPath + "\"\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake %s: %v", name, err)
	}
}

func TestFocusLinuxRunsApplicableTools(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	for _, name := range []string{"tmux", "kitty", "wezterm", "code", "xdotool"} {
		writeFakeTool(t, dir, name, logPath)
	}
	t.Setenv("PATH", dir)

	target := card.FocusTarget{
		TmuxSocket:    "/tmp/sock",
		TmuxPane:      "%3",
		KittyListenOn: "unix:/tmp/kitty",
		KittyWindowID: "7",
		WeztermPane:   "9",
		Terminal:      "vscode",
		CWD:           "/work",
		X11WindowID:   "12345",
	}
	if err := focusLinux(target); err != nil {
		t.Fatalf("focusLinux() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	log := string(data)

	for _, want := range []string{
		"tmux -S /tmp/sock select-pane -t %3",
		"kitty @ --to unix:/tmp/kitty focus-window --match id:7",
		"wezterm cli activate-pane --pane-id 9",
		"code --reuse-window /work",
		"xdotool windowactivate 12345",
	} {
		if !strings.Contains(log, want) {
			t.Errorf("call log missing %q; got:\n%s", want, log)
		}
	}
}

func TestFocusLinuxCursorRunsCursorCLI(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeTool(t, dir, "cursor", logPath)
	writeFakeTool(t, dir, "code", logPath)
	t.Setenv("PATH", dir)

	target := card.FocusTarget{Terminal: "vscode", AppName: "Cursor", CWD: "/work"}
	if err := focusLinux(target); err != nil {
		t.Fatalf("focusLinux() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	log := string(data)
	if !strings.Contains(log, "cursor --reuse-window /work") {
		t.Errorf("call log missing cursor invocation; got:\n%s", log)
	}
	if strings.Contains(log, "/code ") {
		t.Errorf("cursor on PATH should not fall back to code; got:\n%s", log)
	}
}

func TestFocusLinuxCursorFallsBackToCodeWhenCursorMissing(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeTool(t, dir, "code", logPath) // no cursor on PATH
	t.Setenv("PATH", dir)

	target := card.FocusTarget{Terminal: "vscode", AppName: "Cursor", CWD: "/work"}
	if err := focusLinux(target); err != nil {
		t.Fatalf("focusLinux() error = %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	if !strings.Contains(string(data), "code --reuse-window /work") {
		t.Errorf("call log = %q, want a code fallback call", data)
	}
}

func TestFocusLinuxMissingToolsNeverFail(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no tools on PATH at all
	target := card.FocusTarget{
		TmuxSocket:  "/tmp/sock",
		TmuxPane:    "%3",
		X11WindowID: "12345",
	}
	if err := focusLinux(target); err != nil {
		t.Fatalf("focusLinux() with no tools installed should not error, got %v", err)
	}
}

func TestFocusLinuxSkipsUnsetTargets(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeTool(t, dir, "tmux", logPath)
	writeFakeTool(t, dir, "xdotool", logPath)
	t.Setenv("PATH", dir)

	// No tmux pane, no X11 window id: nothing should run, so the log stays untouched.
	if err := focusLinux(card.FocusTarget{}); err != nil {
		t.Fatalf("focusLinux() error = %v", err)
	}
	if _, err := os.Stat(logPath); err == nil {
		t.Fatal("expected no tool to run when target fields are unset")
	}
}

func TestFocusLinuxFallsBackToWmctrl(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeTool(t, dir, "wmctrl", logPath) // no xdotool on PATH
	t.Setenv("PATH", dir)

	if err := focusLinux(card.FocusTarget{X11WindowID: "255"}); err != nil {
		t.Fatalf("focusLinux() error = %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	if !strings.Contains(string(data), "wmctrl -i -a 0xff") {
		t.Fatalf("call log = %q, want a wmctrl -i -a 0xff call", data)
	}
}

func TestWmctrlID(t *testing.T) {
	if got := wmctrlID("255"); got != "0xff" {
		t.Fatalf("wmctrlID(255) = %q, want 0xff", got)
	}
	if got := wmctrlID("not-a-number"); got != "not-a-number" {
		t.Fatalf("wmctrlID(invalid) = %q, want passthrough", got)
	}
}

func TestIsVSCodeLike(t *testing.T) {
	if !isVSCodeLike("vscode") || !isVSCodeLike("cursor") {
		t.Fatal("vscode and cursor should be recognized")
	}
	if isVSCodeLike("iTerm.app") {
		t.Fatal("iTerm.app should not be recognized as VS Code like")
	}
}
