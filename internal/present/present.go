// Package present shows a Card with the platform's own notification mechanism.
package present

import "github.com/bahri-hirfanoglu/cla-notify/internal/card"

// Presenter shows one card; Show must return quickly because hooks have a 5 second budget.
type Presenter interface {
	// Name identifies the mechanism, e.g. "macos-hud", "linux-dbus", "windows-toast".
	Name() string
	// Check reports why this machine cannot show cards, or nil when it can.
	Check() error
	// Show hands the card to a detached process or the OS and returns without waiting for a click.
	Show(c *card.Card) error
}

// ForOS returns the presenter for the running platform; each platform file sets it in init.
func ForOS() Presenter { return platform }

var platform Presenter = noop{}

type noop struct{}

func (noop) Name() string          { return "none" }
func (noop) Check() error          { return errUnsupported }
func (noop) Show(*card.Card) error { return errUnsupported }
