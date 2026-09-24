//go:build linux

// Linux terminal detection: WINDOWID and a terminal pid climbed from /proc, past tmux when present.
package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func init() {
	addPlatformDetails = addLinuxDetails
}

// terminalProcessNames are the emulators climbToTerminal recognizes as the outer window owner.
var terminalProcessNames = map[string]bool{
	"gnome-terminal-server": true,
	"gnome-terminal":        true,
	"konsole":               true,
	"xterm":                 true,
	"alacritty":             true,
	"kitty":                 true,
	"foot":                  true,
	"xfce4-terminal":        true,
	"terminator":            true,
	"tilix":                 true,
	"urxvt":                 true,
	"st":                    true,
	"terminology":           true,
	"deepin-terminal":       true,
	"lxterminal":            true,
	"mate-terminal":         true,
	"qterminal":             true,
	"wezterm-gui":           true,
	"wezterm":               true,
}

func addLinuxDetails(t *card.FocusTarget, env map[string]string) {
	if id := env["WINDOWID"]; id != "" {
		t.X11WindowID = id
	}
	if pid := findTerminalPID(env); pid > 0 {
		t.TerminalPID = pid
	}
}

// findTerminalPID climbs the process tree from this process, or from the tmux client when inside tmux.
func findTerminalPID(env map[string]string) int {
	start := os.Getpid()
	if env["TMUX"] != "" {
		if cpid := tmuxClientPID(env["TMUX"]); cpid > 0 {
			start = cpid
		}
	}
	return climbToTerminal(start, procInfo)
}

// tmuxClientPID asks tmux itself for the attached client's pid: the server is reparented to init on
// detach, so only the client keeps a real parent link back to the outer terminal.
func tmuxClientPID(tmuxVar string) int {
	socket := strings.Split(tmuxVar, ",")[0]
	if socket == "" {
		return 0
	}
	out, err := exec.Command("tmux", "-S", socket, "list-clients", "-F", "#{client_pid}").Output()
	if err != nil {
		return 0
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	pid, err := strconv.Atoi(line)
	if err != nil {
		return 0
	}
	return pid
}

// climbToTerminal walks parent pids via lookup until it finds a known terminal emulator name, or
// falls back to the topmost ancestor it could still read when none matched.
func climbToTerminal(pid int, lookup func(int) (string, int, error)) int {
	seen := map[int]bool{}
	current := pid
	last := 0
	for current > 1 && !seen[current] {
		seen[current] = true
		name, ppid, err := lookup(current)
		if err != nil {
			break
		}
		if terminalProcessNames[name] {
			return current
		}
		last = current
		current = ppid
	}
	return last
}

// procInfo reads a process's command name and parent pid from /proc/<pid>/stat.
func procInfo(pid int) (name string, ppid int, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", 0, err
	}
	s := string(data)
	open := strings.IndexByte(s, '(')
	closeParen := strings.LastIndexByte(s, ')')
	if open < 0 || closeParen < 0 || closeParen < open {
		return "", 0, fmt.Errorf("malformed stat for pid %d", pid)
	}
	name = s[open+1 : closeParen]
	rest := strings.Fields(s[closeParen+1:])
	if len(rest) < 2 {
		return "", 0, fmt.Errorf("short stat for pid %d", pid)
	}
	ppid, err = strconv.Atoi(rest[1])
	return name, ppid, err
}
