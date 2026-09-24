package present

import "path/filepath"

// stateDir resolves the state directory from CLA_NOTIFY_STATE_DIR, else %LOCALAPPDATA%\cla-notify.
func stateDir(getenv func(string) string) string {
	if v := getenv("CLA_NOTIFY_STATE_DIR"); v != "" {
		return v
	}
	return filepath.Join(getenv("LOCALAPPDATA"), "cla-notify")
}

// cardsDir is where every presenter writes the cards a click later reads back.
func cardsDir(state string) string { return filepath.Join(state, "cards") }

// cardFilePath is the on-disk path for one session's card.
func cardFilePath(state, id string) string { return filepath.Join(cardsDir(state), id+".json") }
