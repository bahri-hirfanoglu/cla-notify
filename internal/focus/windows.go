//go:build windows

// Package focus, windows: raises the terminal window a card's session ran in.
package focus

import (
	"encoding/json"
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func init() {
	impl = windowsFocus
}

var errWindowNotFound = errors.New("cla-notify: no terminal window to focus")

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
)

const swRestore = 9

// RunURL validates a cla-notify://focus?card=<path> URL, focuses the terminal it names, and
// deletes the card file; the windows toast protocol handler and action button call this.
func RunURL(raw string) error {
	state := stateDir(os.Getenv)
	path, err := parseFocusURL(raw, cardsDir(state))
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var c card.Card
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}

	focusErr := windowsFocus(c.Focus)
	_ = os.Remove(path)
	return focusErr
}

func windowsFocus(t card.FocusTarget) error {
	if t.Terminal == "vscode" {
		return runVSCodeLike(t)
	}

	if hwnd, ok := parseHWND(t.WindowsHWND); ok {
		return raiseWindow(windows.HWND(hwnd))
	}

	if t.TerminalPID != 0 {
		if hwnd, ok := findTopLevelWindow(uint32(t.TerminalPID)); ok {
			return raiseWindow(hwnd)
		}
	}

	return errWindowNotFound
}

// raiseWindow restores and foregrounds hwnd, using the AttachThreadInput dance Windows
// requires to let a background process steal the foreground.
func raiseWindow(hwnd windows.HWND) error {
	if hwnd == 0 || !windows.IsWindow(hwnd) {
		return errWindowNotFound
	}

	procShowWindow.Call(uintptr(hwnd), swRestore)

	var targetPID uint32
	targetTID, _ := windows.GetWindowThreadProcessId(hwnd, &targetPID)
	procAllowSetForegroundWindow.Call(uintptr(targetPID))

	fg, _, _ := procGetForegroundWindow.Call()
	if fg != 0 {
		var fgPID uint32
		fgTID, _ := windows.GetWindowThreadProcessId(windows.HWND(fg), &fgPID)
		currentTID := windows.GetCurrentThreadId()
		if fgTID != 0 && fgTID != targetTID {
			procAttachThreadInput.Call(uintptr(currentTID), uintptr(fgTID), 1)
			defer procAttachThreadInput.Call(uintptr(currentTID), uintptr(fgTID), 0)
		}
	}

	r, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	if r == 0 {
		return errWindowNotFound
	}
	return nil
}

// findTopLevelWindow returns the first visible top-level window owned by pid.
func findTopLevelWindow(pid uint32) (windows.HWND, bool) {
	var found windows.HWND
	cb := syscall.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		var winPID uint32
		windows.GetWindowThreadProcessId(hwnd, &winPID)
		if winPID == pid && windows.IsWindowVisible(hwnd) {
			found = hwnd
			return 0
		}
		return 1
	})
	windows.EnumWindows(cb, nil)
	return found, found != 0
}
