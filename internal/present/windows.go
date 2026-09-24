//go:build windows

// Package present, windows: shows cards as WinRT toast notifications.
package present

import (
	"encoding/json"
	"os"
	"time"

	"github.com/go-ole/go-ole"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func init() {
	platform = windowsToast{}
}

type windowsToast struct{}

func (windowsToast) Name() string { return "windows-toast" }

// Check reports whether the WinRT toast notifier can be loaded on this machine.
func (windowsToast) Check() error {
	if err := ensureWinRT(); err != nil {
		return err
	}
	insp, err := ole.RoGetActivationFactory(rtToastNotificationManager, iidIToastNotificationManagerStatics)
	if err != nil {
		return err
	}
	insp.Release()
	return nil
}

// Show writes c to the state dir and shows it as a WinRT toast; it returns as soon as the
// notifier has accepted the toast, without waiting for a click.
func (windowsToast) Show(c *card.Card) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	state := stateDir(os.Getenv)
	if err := ensureRegistration(exePath, state); err != nil {
		return err
	}

	dir := cardsDir(state)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := cardFilePath(state, c.SessionID)
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}

	targetURL := focusURL(path)
	xmlDoc := buildToastXML(c, targetURL)
	expires := c.CreatedAt.Add(time.Duration(c.DurationSecs * float64(time.Second)))

	return showToast(xmlDoc, aumid, toastTag(c.SessionID), toastGroup, expires)
}
