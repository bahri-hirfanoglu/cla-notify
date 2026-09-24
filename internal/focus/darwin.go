//go:build darwin

package focus

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/paths"
)

func init() {
	impl = darwinFocus
}

// darwinFocus shells out to "cla-notify-hud focus <card.json>" so `cla-notify focus` works on every OS,
// even though the HUD normally handles clicks itself without ever invoking this command.
func darwinFocus(t card.FocusTarget) error {
	hud, err := hudPath()
	if err != nil {
		return err
	}

	stateDir, err := paths.StateDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(stateDir, "focus")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	id, err := newFocusID()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, id+".json")

	// The HUD decodes a complete card, so every required field gets a valid value.
	data, err := json.Marshal(card.Card{Version: card.Version, Kind: card.KindDone, Options: []string{}, Focus: t, CreatedAt: time.Now()})
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	defer os.Remove(path)

	return exec.Command(hud, "focus", path).Run()
}

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

func newFocusID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b[:]), nil
}
