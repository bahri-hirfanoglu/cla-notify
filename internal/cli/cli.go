// Package cli implements the cla-notify subcommands.
package cli

import (
	"fmt"
	"io"
)

// Version is the cla-notify release version.
var Version = "0.0.2"

// Run dispatches args[0] to a subcommand and returns the process exit code.
// 0 is ok, 1 is a validation or usage-level failure the command reported, 64 is bad usage.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cla-notify <hook|test|config|doctor|focus|version>")
		return 64
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "hook":
		return runHook(stdin, stdout)
	case "test":
		return runTest(rest, stdout, stderr)
	case "config":
		return runConfig(rest, stdout, stderr)
	case "doctor":
		return runDoctor(stdout)
	case "focus":
		return runFocus(rest, stderr)
	case "version":
		return runVersion(stdout)
	case "present-linux":
		return runPresentLinux(rest, stderr)
	case "focus-url":
		return runFocusURL(rest, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		return 64
	}
}
