//go:build windows

// Package terminal, windows: finds the terminal PID and top-level window via the process tree.
package terminal

import (
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func init() {
	addPlatformDetails = addWindowsDetails
}

func addWindowsDetails(t *card.FocusTarget, env map[string]string) {
	if wt := env["WT_SESSION"]; wt != "" && t.WTSession == "" {
		t.WTSession = wt
	}

	snapshot, err := processSnapshot()
	if err != nil {
		return
	}

	owner, hwnd, ok := walkToWindowOwner(snapshot, windows.GetCurrentProcessId(), hasVisibleWindow)
	if !ok {
		return
	}
	t.TerminalPID = int(owner.PID)
	t.WindowsHWND = strconv.FormatUint(uint64(hwnd), 10)
	if t.AppName == "" {
		t.AppName = appNameForExe(owner.Exe)
	}
}

// processSnapshot walks the whole process table via Toolhelp32, the standard way to read
// a process's ancestry on Windows.
func processSnapshot() ([]ProcEntry, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)

	var entries []ProcEntry
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return nil, err
	}
	for {
		entries = append(entries, ProcEntry{
			PID:       pe.ProcessID,
			ParentPID: pe.ParentProcessID,
			Exe:       windows.UTF16ToString(pe.ExeFile[:]),
		})
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return entries, nil
}

// hasVisibleWindow reports whether pid owns a visible top-level window, and its handle.
func hasVisibleWindow(pid uint32) (uintptr, bool) {
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
	return uintptr(found), found != 0
}
