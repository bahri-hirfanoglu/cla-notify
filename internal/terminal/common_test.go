package terminal

import (
	"testing"
)

func TestDetectCommon(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantTerm   string
		wantApp    string
		wantBundle string
	}{
		{
			name:       "iTerm2",
			env:        map[string]string{"TERM_PROGRAM": "iTerm.app", "ITERM_SESSION_ID": "w0t0p0:ABCD-1234"},
			wantTerm:   "iTerm.app",
			wantApp:    "iTerm2",
			wantBundle: "com.googlecode.iterm2",
		},
		{
			name:       "Apple Terminal",
			env:        map[string]string{"TERM_PROGRAM": "Apple_Terminal"},
			wantTerm:   "Apple_Terminal",
			wantApp:    "Terminal",
			wantBundle: "com.apple.Terminal",
		},
		{
			name:       "VS Code",
			env:        map[string]string{"TERM_PROGRAM": "vscode"},
			wantTerm:   "vscode",
			wantApp:    "Visual Studio Code",
			wantBundle: "com.microsoft.VSCode",
		},
		{
			name:       "Cursor via bundle id",
			env:        map[string]string{"TERM_PROGRAM": "vscode", "__CFBundleIdentifier": "com.todesktop.230313mzl4w4u92"},
			wantTerm:   "vscode",
			wantApp:    "Cursor",
			wantBundle: "com.todesktop.230313mzl4w4u92",
		},
		{
			name:       "Cursor via trace id, no TERM_PROGRAM",
			env:        map[string]string{"CURSOR_TRACE_ID": "abc123"},
			wantTerm:   "vscode",
			wantApp:    "Cursor",
			wantBundle: "com.todesktop.230313mzl4w4u92",
		},
		{
			name:       "Windows Terminal",
			env:        map[string]string{"WT_SESSION": "abc-123"},
			wantTerm:   "WindowsTerminal",
			wantApp:    "Windows Terminal",
			wantBundle: "",
		},
		{
			name:       "ghostty",
			env:        map[string]string{"TERM_PROGRAM": "ghostty"},
			wantTerm:   "ghostty",
			wantApp:    "Ghostty",
			wantBundle: "com.mitchellh.ghostty",
		},
		{
			name:       "Warp",
			env:        map[string]string{"TERM_PROGRAM": "WarpTerminal"},
			wantTerm:   "WarpTerminal",
			wantApp:    "Warp",
			wantBundle: "dev.warp.Warp-Stable",
		},
		{
			name:       "WezTerm",
			env:        map[string]string{"TERM_PROGRAM": "WezTerm", "WEZTERM_PANE": "3"},
			wantTerm:   "WezTerm",
			wantApp:    "WezTerm",
			wantBundle: "com.github.wez.wezterm",
		},
		{
			name:       "kitty",
			env:        map[string]string{"TERM_PROGRAM": "kitty", "KITTY_WINDOW_ID": "1", "KITTY_LISTEN_ON": "unix:/tmp/x"},
			wantTerm:   "kitty",
			wantApp:    "kitty",
			wantBundle: "net.kovidgoyal.kitty",
		},
		{
			name:       "unknown empty environment",
			env:        map[string]string{},
			wantTerm:   "unknown",
			wantApp:    "Terminal",
			wantBundle: "",
		},
		{
			name:       "explicit bundle id overrides the guessed default",
			env:        map[string]string{"TERM_PROGRAM": "iTerm.app", "__CFBundleIdentifier": "com.custom.fork"},
			wantTerm:   "iTerm.app",
			wantApp:    "iTerm2",
			wantBundle: "com.custom.fork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCommon("/repo", tt.env)
			if got.Terminal != tt.wantTerm {
				t.Errorf("Terminal = %q, want %q", got.Terminal, tt.wantTerm)
			}
			if got.AppName != tt.wantApp {
				t.Errorf("AppName = %q, want %q", got.AppName, tt.wantApp)
			}
			if got.BundleID != tt.wantBundle {
				t.Errorf("BundleID = %q, want %q", got.BundleID, tt.wantBundle)
			}
			if got.CWD != "/repo" {
				t.Errorf("CWD = %q, want /repo", got.CWD)
			}
		})
	}
}

func TestDetectCommonItermSessionID(t *testing.T) {
	got := detectCommon("/repo", map[string]string{"TERM_PROGRAM": "iTerm.app", "ITERM_SESSION_ID": "w0t0p0:ABCD-1234"})
	if got.ItermSessionID != "ABCD-1234" {
		t.Errorf("ItermSessionID = %q, want ABCD-1234", got.ItermSessionID)
	}
}

func TestDetectCommonTmux(t *testing.T) {
	got := detectCommon("/repo", map[string]string{
		"TERM_PROGRAM": "tmux", "TMUX": "/tmp/tmux-1000/default,1234,0", "TMUX_PANE": "%3",
	})
	if got.TmuxSocket != "/tmp/tmux-1000/default" {
		t.Errorf("TmuxSocket = %q", got.TmuxSocket)
	}
	if got.TmuxPane != "%3" {
		t.Errorf("TmuxPane = %q, want %%3", got.TmuxPane)
	}
	if got.Terminal != "tmux" {
		t.Errorf("Terminal = %q, want tmux (platform code resolves the outer terminal)", got.Terminal)
	}
}

func TestDetectCommonX11WindowID(t *testing.T) {
	got := detectCommon("/repo", map[string]string{"WINDOWID": "12345"})
	if got.X11WindowID != "12345" {
		t.Errorf("X11WindowID = %q, want 12345", got.X11WindowID)
	}
}

func TestDetect(t *testing.T) {
	got := Detect("/repo", map[string]string{"TERM_PROGRAM": "iTerm.app"})
	if got.CWD != "/repo" || got.Terminal != "iTerm.app" {
		t.Errorf("Detect() = %+v, unexpected", got)
	}
}
