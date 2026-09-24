package focus

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func writeFakeCLI(t *testing.T, dir, name, logPath string) {
	t.Helper()
	script := "#!/bin/sh\necho \"$0 $*\" >> \"" + logPath + "\"\n"
	path := filepath.Join(dir, name)
	// Windows cannot run a shell script, so the fake is a batch file found through PATHEXT.
	if runtime.GOOS == "windows" {
		script = "@echo %~n0 %*>> \"" + logPath + "\"\r\n"
		path += ".bat"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake %s: %v", name, err)
	}
}

func TestRunVSCodeLikeCursorOnPath(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeCLI(t, dir, "cursor", logPath)
	writeFakeCLI(t, dir, "code", logPath)
	t.Setenv("PATH", dir)

	if err := runVSCodeLike(card.FocusTarget{AppName: "Cursor", CWD: "/work"}); err != nil {
		t.Fatalf("runVSCodeLike() error = %v", err)
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

func TestRunVSCodeLikeCursorMissingFallsBackToCode(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeCLI(t, dir, "code", logPath) // no cursor on PATH
	t.Setenv("PATH", dir)

	if err := runVSCodeLike(card.FocusTarget{AppName: "Cursor", CWD: "/work"}); err != nil {
		t.Fatalf("runVSCodeLike() error = %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	if !strings.Contains(string(data), "code --reuse-window /work") {
		t.Errorf("call log = %q, want a code fallback call", data)
	}
}

func TestRunVSCodeLikePlainVSCodeUsesCode(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	writeFakeCLI(t, dir, "cursor", logPath)
	writeFakeCLI(t, dir, "code", logPath)
	t.Setenv("PATH", dir)

	if err := runVSCodeLike(card.FocusTarget{AppName: "Visual Studio Code", CWD: "/work"}); err != nil {
		t.Fatalf("runVSCodeLike() error = %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read call log: %v", err)
	}
	if !strings.Contains(string(data), "code --reuse-window /work") {
		t.Errorf("call log = %q, want a code call", data)
	}
}

func TestIsCursor(t *testing.T) {
	if !isCursor(card.FocusTarget{AppName: "Cursor"}) {
		t.Error("AppName Cursor should be recognized")
	}
	if !isCursor(card.FocusTarget{BundleID: cursorBundleID}) {
		t.Error("Cursor bundle id should be recognized")
	}
	if isCursor(card.FocusTarget{AppName: "Visual Studio Code"}) {
		t.Error("plain VS Code should not be recognized as Cursor")
	}
}
