//go:build linux

// Linux focus: multiplexer/terminal CLIs first, then an X11 window raise.
package focus

import (
	"os/exec"
	"strconv"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func init() {
	impl = focusLinux
}

// focusLinux tries every applicable tool in turn; a missing tool or a failed call is never an error.
func focusLinux(t card.FocusTarget) error {
	if t.TmuxPane != "" && toolExists("tmux") {
		_ = focusTmux(t)
	}
	if t.KittyListenOn != "" && t.KittyWindowID != "" && toolExists("kitty") {
		_ = run("kitty", "@", "--to", t.KittyListenOn, "focus-window", "--match", "id:"+t.KittyWindowID)
	}
	if t.WeztermPane != "" && toolExists("wezterm") {
		_ = run("wezterm", "cli", "activate-pane", "--pane-id", t.WeztermPane)
	}
	if isVSCodeLike(t.Terminal) && t.CWD != "" {
		if isCursor(t) && toolExists("cursor") {
			_ = run("cursor", "--reuse-window", t.CWD)
		} else if toolExists("code") {
			_ = run("code", "--reuse-window", t.CWD)
		}
	}
	if t.X11WindowID != "" {
		raiseX11(t.X11WindowID)
	}
	return nil
}

func focusTmux(t card.FocusTarget) error {
	args := []string{}
	if t.TmuxSocket != "" {
		args = append(args, "-S", t.TmuxSocket)
	}
	args = append(args, "select-pane", "-t", t.TmuxPane)
	return run("tmux", args...)
}

func isVSCodeLike(terminal string) bool {
	switch terminal {
	case "vscode", "cursor":
		return true
	default:
		return false
	}
}

// raiseX11 tries xdotool then wmctrl; on Wayland (no X11WindowID) this is never called, so it skips cleanly.
func raiseX11(id string) {
	if toolExists("xdotool") {
		_ = run("xdotool", "windowactivate", id)
		return
	}
	if toolExists("wmctrl") {
		_ = run("wmctrl", "-i", "-a", wmctrlID(id))
	}
}

// wmctrlID renders a decimal WINDOWID as the hex form wmctrl -i expects, falling back to the input as-is.
func wmctrlID(id string) string {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return id
	}
	return "0x" + strconv.FormatInt(n, 16)
}

func toolExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}
