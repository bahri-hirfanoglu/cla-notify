package present

import "path/filepath"

// shortcutName is the Start Menu shortcut file that carries cla-notify's AppUserModelID.
const shortcutName = "cla-notify.lnk"

// startMenuProgramsDir is the per-user Start Menu Programs folder; no admin rights needed.
func startMenuProgramsDir(getenv func(string) string) string {
	return filepath.Join(getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs")
}

// shortcutPath is where cla-notify.lnk lives.
func shortcutPath(getenv func(string) string) string {
	return filepath.Join(startMenuProgramsDir(getenv), shortcutName)
}
