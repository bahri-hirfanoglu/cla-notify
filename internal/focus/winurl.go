// Package focus: pure cla-notify:// URL validation for the Windows click path, tested on every OS.
package focus

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

var errInvalidFocusURL = errors.New("invalid cla-notify focus URL")

// stateDir resolves the state directory the same way the windows presenter does.
func stateDir(getenv func(string) string) string {
	if v := getenv("CLA_NOTIFY_STATE_DIR"); v != "" {
		return v
	}
	return filepath.Join(getenv("LOCALAPPDATA"), "cla-notify")
}

// cardsDir is where a click's card file is expected to live.
func cardsDir(state string) string { return filepath.Join(state, "cards") }

// parseFocusURL validates a cla-notify://focus?card=<path> URL and returns the card path,
// rejecting anything whose path falls outside cardsDirPath.
func parseFocusURL(raw, cardsDirPath string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errInvalidFocusURL
	}
	if u.Scheme != "cla-notify" || u.Host != "focus" {
		return "", errInvalidFocusURL
	}
	cardParam := u.Query().Get("card")
	if cardParam == "" {
		return "", errInvalidFocusURL
	}

	clean := filepath.Clean(cardParam)
	cardsClean := filepath.Clean(cardsDirPath)
	if !strings.HasPrefix(clean, cardsClean+string(filepath.Separator)) {
		return "", errInvalidFocusURL
	}
	return clean, nil
}
