package terminal

import (
	"strings"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

const cursorBundleID = "com.todesktop.230313mzl4w4u92"

// detectCommon reads the portable environment signals every terminal sets; platform code may refine the result.
func detectCommon(cwd string, env map[string]string) card.FocusTarget {
	termProgram := env["TERM_PROGRAM"]

	var itermSessionID string
	if sid := env["ITERM_SESSION_ID"]; sid != "" {
		if idx := strings.Index(sid, ":"); idx >= 0 {
			itermSessionID = sid[idx+1:]
		} else {
			itermSessionID = sid
		}
	}

	var tmuxPane, tmuxSocket string
	if tmux := env["TMUX"]; tmux != "" {
		tmuxPane = env["TMUX_PANE"]
		if idx := strings.Index(tmux, ","); idx >= 0 {
			tmuxSocket = tmux[:idx]
		} else {
			tmuxSocket = tmux
		}
	}

	isCursor := env["__CFBundleIdentifier"] == cursorBundleID || env["CURSOR_TRACE_ID"] != ""

	terminal := termProgram
	appName := termProgram
	bundleID := env["__CFBundleIdentifier"]

	switch termProgram {
	case "iTerm.app":
		appName = "iTerm2"
		bundleID = orDefault(bundleID, "com.googlecode.iterm2")
	case "Apple_Terminal":
		appName = "Terminal"
		bundleID = orDefault(bundleID, "com.apple.Terminal")
	case "vscode":
		if isCursor {
			appName = "Cursor"
			bundleID = orDefault(bundleID, cursorBundleID)
		} else {
			appName = "Visual Studio Code"
			bundleID = orDefault(bundleID, "com.microsoft.VSCode")
		}
	case "ghostty":
		appName = "Ghostty"
		bundleID = orDefault(bundleID, "com.mitchellh.ghostty")
	case "WarpTerminal":
		appName = "Warp"
		bundleID = orDefault(bundleID, "dev.warp.Warp-Stable")
	case "WezTerm":
		appName = "WezTerm"
		bundleID = orDefault(bundleID, "com.github.wez.wezterm")
	case "kitty":
		appName = "kitty"
		bundleID = orDefault(bundleID, "net.kovidgoyal.kitty")
	case "":
		switch {
		case env["WT_SESSION"] != "":
			terminal = "WindowsTerminal"
			appName = "Windows Terminal"
		case isCursor:
			terminal = "vscode"
			appName = "Cursor"
			bundleID = orDefault(bundleID, cursorBundleID)
		case env["VSCODE_PID"] != "" || env["VSCODE_INJECTION"] != "":
			terminal = "vscode"
			appName = "Visual Studio Code"
			bundleID = orDefault(bundleID, "com.microsoft.VSCode")
		default:
			appName = "Terminal"
		}
	}

	if terminal == "" {
		terminal = "unknown"
	}
	if appName == "" {
		appName = "Terminal"
	}

	return card.FocusTarget{
		Terminal:       terminal,
		AppName:        appName,
		BundleID:       bundleID,
		ItermSessionID: itermSessionID,
		TmuxPane:       tmuxPane,
		TmuxSocket:     tmuxSocket,
		WeztermPane:    env["WEZTERM_PANE"],
		KittyWindowID:  env["KITTY_WINDOW_ID"],
		KittyListenOn:  env["KITTY_LISTEN_ON"],
		X11WindowID:    env["WINDOWID"],
		WTSession:      env["WT_SESSION"],
		CWD:            cwd,
	}
}

func orDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
