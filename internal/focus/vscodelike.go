package focus

import (
	"os/exec"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

// runVSCodeLike launches Cursor's own CLI when the target is Cursor and it is on PATH, else falls
// back to VS Code's; only windows.go calls it, but it has no build tag so it is testable on every host.
func runVSCodeLike(t card.FocusTarget) error {
	if isCursor(t) {
		if _, err := exec.LookPath("cursor"); err == nil {
			return exec.Command("cursor", "--reuse-window", t.CWD).Run()
		}
	}
	return exec.Command("code", "--reuse-window", t.CWD).Run()
}
