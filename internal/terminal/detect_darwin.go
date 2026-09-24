//go:build darwin

package terminal

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func init() {
	addPlatformDetails = addDarwinDetails
}

// knownProcessNames maps a process name fragment to the TERM_PROGRAM it implies, used to find the
// outer terminal behind tmux, which hides it from the environment.
var knownProcessNames = []struct{ match, termProgram string }{
	{"iTerm2", "iTerm.app"},
	{"Terminal", "Apple_Terminal"},
	{"Cursor Helper", "vscode"},
	{"Code Helper", "vscode"},
	{"ghostty", "ghostty"},
	{"Warp", "WarpTerminal"},
	{"wezterm-gui", "WezTerm"},
	{"kitty", "kitty"},
}

func addDarwinDetails(t *card.FocusTarget, env map[string]string) {
	t.TTY = ttyFromProcessTree()

	if t.Terminal != "tmux" {
		return
	}
	outer := outerTerminalFromProcessTree()
	if outer == "" {
		return
	}
	t.Terminal = outer
	switch outer {
	case "iTerm.app":
		t.AppName, t.BundleID = "iTerm2", "com.googlecode.iterm2"
	case "Apple_Terminal":
		t.AppName, t.BundleID = "Terminal", "com.apple.Terminal"
	case "WarpTerminal":
		t.AppName, t.BundleID = "Warp", "dev.warp.Warp-Stable"
	case "WezTerm":
		t.AppName, t.BundleID = "WezTerm", "com.github.wez.wezterm"
	case "kitty":
		t.AppName, t.BundleID = "kitty", "net.kovidgoyal.kitty"
	case "ghostty":
		t.AppName, t.BundleID = "Ghostty", "com.mitchellh.ghostty"
	case "vscode":
		t.AppName, t.BundleID = "Visual Studio Code", "com.microsoft.VSCode"
	}
}

// ttyFromProcessTree climbs our own ppid chain until a process with a tty is found ("??" means none).
func ttyFromProcessTree() string {
	pid := os.Getpid()
	for i := 0; i < 10; i++ {
		if tty := processTTY(pid); tty != "" && tty != "??" {
			return "/dev/" + tty
		}
		parent := parentPid(pid)
		if parent <= 1 {
			break
		}
		pid = parent
	}
	return ""
}

// outerTerminalFromProcessTree climbs the ppid chain looking for a known terminal app's process name.
func outerTerminalFromProcessTree() string {
	pid := os.Getpid()
	for i := 0; i < 12; i++ {
		parent := parentPid(pid)
		if parent <= 1 {
			break
		}
		pid = parent
		name := processName(pid)
		for _, candidate := range knownProcessNames {
			if strings.Contains(name, candidate.match) {
				return candidate.termProgram
			}
		}
	}
	return ""
}

func processTTY(pid int) string {
	out, err := exec.Command("/bin/ps", "-o", "tty=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func processName(pid int) string {
	out, err := exec.Command("/bin/ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parentPid(pid int) int {
	out, err := exec.Command("/bin/ps", "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0
	}
	p, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return p
}
