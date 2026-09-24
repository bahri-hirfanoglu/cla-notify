package terminal

import "testing"

func TestWalkToWindowOwnerFindsAncestor(t *testing.T) {
	snapshot := []ProcEntry{
		{PID: 1, ParentPID: 0, Exe: "explorer.exe"},
		{PID: 10, ParentPID: 1, Exe: "WindowsTerminal.exe"},
		{PID: 20, ParentPID: 10, Exe: "pwsh.exe"},
		{PID: 30, ParentPID: 20, Exe: "node.exe"},
		{PID: 40, ParentPID: 30, Exe: "cla-notify.exe"},
	}
	hasWindow := func(pid uint32) (uintptr, bool) {
		if pid == 10 {
			return 0xABCD, true
		}
		return 0, false
	}

	owner, hwnd, ok := walkToWindowOwner(snapshot, 40, hasWindow)
	if !ok {
		t.Fatal("walkToWindowOwner() did not find an owner")
	}
	if owner.PID != 10 || hwnd != 0xABCD {
		t.Fatalf("walkToWindowOwner() = %+v, %#x, want pid 10, 0xabcd", owner, hwnd)
	}
}

func TestWalkToWindowOwnerNoMatch(t *testing.T) {
	snapshot := []ProcEntry{
		{PID: 1, ParentPID: 0, Exe: "explorer.exe"},
		{PID: 40, ParentPID: 1, Exe: "cla-notify.exe"},
	}
	_, _, ok := walkToWindowOwner(snapshot, 40, func(uint32) (uintptr, bool) { return 0, false })
	if ok {
		t.Fatal("walkToWindowOwner() reported a match when none exists")
	}
}

func TestWalkToWindowOwnerStopsOnCycle(t *testing.T) {
	snapshot := []ProcEntry{
		{PID: 1, ParentPID: 2, Exe: "a.exe"},
		{PID: 2, ParentPID: 1, Exe: "b.exe"},
	}
	_, _, ok := walkToWindowOwner(snapshot, 1, func(uint32) (uintptr, bool) { return 0, false })
	if ok {
		t.Fatal("walkToWindowOwner() should not find a match in a cyclic snapshot")
	}
}

func TestWalkToWindowOwnerStopsOnSelfParent(t *testing.T) {
	snapshot := []ProcEntry{{PID: 1, ParentPID: 1, Exe: "init.exe"}}
	_, _, ok := walkToWindowOwner(snapshot, 1, func(uint32) (uintptr, bool) { return 0, false })
	if ok {
		t.Fatal("walkToWindowOwner() should stop when a process is its own parent")
	}
}

func TestAppNameForExe(t *testing.T) {
	cases := map[string]string{
		"WindowsTerminal.exe": "Windows Terminal",
		"Code.exe":            "VS Code",
		"Cursor.exe":          "Cursor",
		"powershell.exe":      "PowerShell",
		"pwsh.exe":            "PowerShell",
		"cmd.exe":             "Command Prompt",
		"mintty.exe":          "mintty",
	}
	for exe, want := range cases {
		if got := appNameForExe(exe); got != want {
			t.Errorf("appNameForExe(%q) = %q, want %q", exe, got, want)
		}
	}
}
