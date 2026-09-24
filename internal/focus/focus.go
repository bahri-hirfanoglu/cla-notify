// Package focus brings the terminal a card came from back to the front.
package focus

import "github.com/bahri-hirfanoglu/cla-notify/internal/card"

// Focus raises the terminal described by t; each platform file sets impl in init.
func Focus(t card.FocusTarget) error { return impl(t) }

var impl = func(card.FocusTarget) error { return nil }

// cursorBundleID mirrors internal/terminal's cursorBundleID; focus avoids importing that package to dodge a cycle.
const cursorBundleID = "com.todesktop.230313mzl4w4u92"

// isCursor reports whether t names the Cursor editor, per however internal/terminal/common.go marked it.
func isCursor(t card.FocusTarget) bool {
	return t.AppName == "Cursor" || t.BundleID == cursorBundleID
}
