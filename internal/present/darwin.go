//go:build darwin

package present

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/paths"
)

func init() {
	platform = darwinPresenter{}
}

type darwinPresenter struct{}

func (darwinPresenter) Name() string { return "macos-hud" }

func (darwinPresenter) Check() error {
	hud, err := hudPath()
	if err != nil {
		return err
	}
	info, err := os.Stat(hud)
	if err != nil {
		return fmt.Errorf("cla-notify-hud not found at %s", hud)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("cla-notify-hud at %s is not executable", hud)
	}
	return nil
}

// Show writes the card next to the state dir and spawns a detached cla-notify-hud to draw it and
// return immediately: hooks have a 5 second budget and must never wait on the presenter.
func (darwinPresenter) Show(c *card.Card) error {
	hud, err := hudPath()
	if err != nil {
		return err
	}

	stateDir, err := paths.StateDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(stateDir, "cards")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	id, err := newUUID()
	if err != nil {
		return err
	}
	cardPath := filepath.Join(dir, id+".json")

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cardPath, data, 0o600); err != nil {
		return err
	}

	cmd := exec.Command(hud, "show", cardPath)
	cmd.Stdin = nil
	cmd.Env = filteredEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = os.Remove(cardPath)
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// hudPath finds cla-notify-hud: an explicit override, else next to this binary, else ../dist next to it.
func hudPath() (string, error) {
	if v := os.Getenv("CLA_NOTIFY_HUD"); v != "" {
		return v, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cla-notify-hud not found (set CLA_NOTIFY_HUD): %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)

	candidate := filepath.Join(dir, "cla-notify-hud")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	candidate = filepath.Join(dir, "..", "dist", "cla-notify-hud")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("cla-notify-hud not found next to %s (set CLA_NOTIFY_HUD)", exe)
}

// filteredEnv keeps only paths and locale for the HUD process: Claude Code's own environment can carry API keys.
func filteredEnv() []string {
	keep := map[string]bool{
		"HOME": true, "PATH": true, "USER": true, "LOGNAME": true,
		"TMPDIR": true, "LANG": true, "XDG_CONFIG_HOME": true,
	}
	var out []string
	for _, kv := range os.Environ() {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		key := kv[:idx]
		if keep[key] || strings.HasPrefix(key, "LC_") || strings.HasPrefix(key, "CLA_NOTIFY_") {
			out = append(out, kv)
		}
	}
	return out
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
