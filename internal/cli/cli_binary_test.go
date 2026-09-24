package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	buildOnce   sync.Once
	binaryPath  string
	buildErr    error
	buildErrOut string
)

// buildBinary compiles cmd/cla-notify once per test run and reuses it across every CLI test.
func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		tmp, err := os.MkdirTemp("", "cla-notify-bin-*")
		if err != nil {
			buildErr = err
			return
		}
		out := filepath.Join(tmp, "cla-notify")
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", out, "github.com/bahri-hirfanoglu/cla-notify/cmd/cla-notify")
		outBytes, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = err
			buildErrOut = string(outBytes)
			return
		}
		binaryPath = out
	})
	if buildErr != nil {
		t.Fatalf("build cla-notify: %v: %s", buildErr, buildErrOut)
	}
	return binaryPath
}

func baseEnv(cfgPath, stateDir string) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"CLA_NOTIFY_CONFIG=" + cfgPath,
		"CLA_NOTIFY_STATE_DIR=" + stateDir,
	}
}

func runCLI(t *testing.T, stdin string, env []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	bin := buildBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return outBuf.String(), errBuf.String(), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("run cla-notify %v: %v (stderr: %s)", args, err, errBuf.String())
	}
	return outBuf.String(), errBuf.String(), 0
}

func newDirs(t *testing.T) (cfgPath, stateDir string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "config.json"), filepath.Join(dir, "state")
}

func TestCLIVersion(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	stdout, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "version")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) != Version {
		t.Errorf("stdout = %q, want %q", stdout, Version)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	_, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "not-a-command")
	if code != 64 {
		t.Errorf("exit code = %d, want 64", code)
	}
}

func TestCLIConfigPath(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	stdout, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "config", "path")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) != cfgPath {
		t.Errorf("stdout = %q, want %q", stdout, cfgPath)
	}
}

func TestCLIConfigInitShowGetSetUnsetValidateReset(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	env := baseEnv(cfgPath, stateDir)

	if _, _, code := runCLI(t, "", env, "config", "init"); code != 0 {
		t.Fatalf("config init exit code = %d, want 0", code)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config init did not create %s: %v", cfgPath, err)
	}

	if _, _, code := runCLI(t, "", env, "config", "init"); code != 1 {
		t.Errorf("config init without --force on an existing file: code = %d, want 1", code)
	}
	if _, _, code := runCLI(t, "", env, "config", "init", "--force"); code != 0 {
		t.Errorf("config init --force: code = %d, want 0", code)
	}

	stdout, _, code := runCLI(t, "", env, "config", "get", "display.corner")
	if code != 0 || strings.TrimSpace(stdout) != "top-right" {
		t.Errorf("config get display.corner = %q (code %d), want top-right", stdout, code)
	}

	// Start from a fresh (never-initialized) file so the next check can tell "set" apart from "init".
	if _, _, code := runCLI(t, "", env, "config", "reset"); code != 0 {
		t.Fatalf("config reset: code = %d, want 0", code)
	}

	if _, stderr, code := runCLI(t, "", env, "config", "set", "display.corner", "bottom-left"); code != 0 {
		t.Fatalf("config set: code = %d, stderr = %s", code, stderr)
	}
	stdout, _, _ = runCLI(t, "", env, "config", "get", "display.corner")
	if strings.TrimSpace(stdout) != "bottom-left" {
		t.Errorf("config get display.corner after set = %q, want bottom-left", stdout)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var tree map[string]interface{}
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	if _, ok := tree["events"]; ok {
		t.Errorf("config set wrote unrelated defaults to disk: %v", tree)
	}

	if _, _, code := runCLI(t, "", env, "config", "unset", "display.corner"); code != 0 {
		t.Errorf("config unset: code = %d, want 0", code)
	}
	stdout, _, _ = runCLI(t, "", env, "config", "get", "display.corner")
	if strings.TrimSpace(stdout) != "top-right" {
		t.Errorf("config get display.corner after unset = %q, want default top-right", stdout)
	}

	if _, _, code := runCLI(t, "", env, "config", "validate"); code != 0 {
		t.Errorf("config validate on a clean file: code = %d, want 0", code)
	}

	if err := os.WriteFile(cfgPath, []byte(`{"nope": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, code = runCLI(t, "", env, "config", "validate")
	if code != 1 || !strings.Contains(stdout, "unknown key: nope") {
		t.Errorf("config validate on a bad file: code=%d stdout=%q", code, stdout)
	}

	if _, _, code := runCLI(t, "", env, "config", "reset"); code != 0 {
		t.Errorf("config reset: code = %d, want 0", code)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config reset should remove %s", cfgPath)
	}
}

func TestCLIConfigShowDefaults(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	stdout, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "config", "show", "--defaults")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &m); err != nil {
		t.Fatalf("config show --defaults did not print valid JSON: %v", err)
	}
	if m["enabled"] != true {
		t.Errorf("config show --defaults: enabled = %v, want true", m["enabled"])
	}
}

func TestCLIHookDryRun(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	env := append(baseEnv(cfgPath, stateDir), "CLA_NOTIFY_DRYRUN=1")
	input := `{
		"session_id": "cli-test",
		"cwd": "/tmp/cli-test-project",
		"hook_event_name": "PreToolUse",
		"tool_name": "AskUserQuestion",
		"tool_input": {"questions": [{"question": "Pick one?", "options": [{"label": "A"}]}]}
	}`
	stdout, _, code := runCLI(t, input, env, "hook")
	if code != 0 {
		t.Fatalf("hook exit code = %d, want 0 (the hook must never fail loudly)", code)
	}
	var c map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &c); err != nil {
		t.Fatalf("hook --dry-run did not print a valid Card: %v (stdout: %s)", err, stdout)
	}
	if c["kind"] != "ask" {
		t.Errorf("Card kind = %v, want ask", c["kind"])
	}
}

func TestCLIHookMalformedInputStillExitsZero(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	_, _, code := runCLI(t, "{not json", baseEnv(cfgPath, stateDir), "hook")
	if code != 0 {
		t.Errorf("hook exit code = %d, want 0 even on malformed input", code)
	}
}

func TestCLITestDryRun(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	stdout, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "test", "done", "--dry-run")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	var c map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &c); err != nil {
		t.Fatalf("test --dry-run did not print a valid Card: %v (stdout: %s)", err, stdout)
	}
	if c["kind"] != "done" {
		t.Errorf("Card kind = %v, want done", c["kind"])
	}
}

func TestCLITestUnknownKind(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	_, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "test", "nope")
	if code != 64 {
		t.Errorf("exit code = %d, want 64", code)
	}
}

func TestCLIDoctor(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	stdout, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "doctor")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "cla-notify doctor") {
		t.Errorf("doctor output = %q, missing header", stdout)
	}
	if !strings.Contains(stdout, stateDir) {
		t.Errorf("doctor output = %q, missing state dir", stdout)
	}
}

func TestCLIFocusMissingFile(t *testing.T) {
	cfgPath, stateDir := newDirs(t)
	_, _, code := runCLI(t, "", baseEnv(cfgPath, stateDir), "focus", filepath.Join(stateDir, "missing.json"))
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
