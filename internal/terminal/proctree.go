package terminal

import "strings"

// ProcEntry is one process in a Toolhelp32 snapshot, or an injected fake for tests.
type ProcEntry struct {
	PID       uint32
	ParentPID uint32
	Exe       string
}

// windowOwner reports whether pid owns a visible top-level window and, if so, its handle.
type windowOwner func(pid uint32) (hwnd uintptr, ok bool)

// walkToWindowOwner walks the parent chain from start, over snapshot, to the first process
// that owns a visible top-level window per hasWindow.
func walkToWindowOwner(snapshot []ProcEntry, start uint32, hasWindow windowOwner) (ProcEntry, uintptr, bool) {
	byPID := make(map[uint32]ProcEntry, len(snapshot))
	for _, p := range snapshot {
		byPID[p.PID] = p
	}

	pid := start
	visited := make(map[uint32]bool, len(snapshot)+1)
	for i := 0; i <= len(snapshot); i++ {
		if pid == 0 || visited[pid] {
			break
		}
		visited[pid] = true

		if hwnd, ok := hasWindow(pid); ok {
			return byPID[pid], hwnd, true
		}

		p, ok := byPID[pid]
		if !ok || p.ParentPID == pid {
			break
		}
		pid = p.ParentPID
	}
	return ProcEntry{}, 0, false
}

// appNameForExe maps a terminal host's executable name to the display name cla-notify uses.
func appNameForExe(exe string) string {
	base := strings.TrimSuffix(strings.ToLower(exe), ".exe")
	switch base {
	case "windowsterminal":
		return "Windows Terminal"
	case "code":
		return "VS Code"
	case "cursor":
		return "Cursor"
	case "powershell", "pwsh":
		return "PowerShell"
	case "cmd":
		return "Command Prompt"
	default:
		return strings.TrimSuffix(exe, ".exe")
	}
}
