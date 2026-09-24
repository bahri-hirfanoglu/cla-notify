package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bahri-hirfanoglu/cla-notify/internal/config"
	"github.com/bahri-hirfanoglu/cla-notify/internal/hook"
)

// parseTestArgs pulls --dry-run out of args wherever it appears; the remaining positional
// argument (if any) selects which kind to show. flag.FlagSet is not used because it stops
// parsing at the first non-flag argument, and "test ask --dry-run" must still work.
func parseTestArgs(args []string) (which string, dryRun bool, err error) {
	which = "all"
	var positional []string
	for _, a := range args {
		switch {
		case a == "--dry-run":
			dryRun = true
		case strings.HasPrefix(a, "-"):
			return "", false, fmt.Errorf("unknown flag: %s", a)
		default:
			positional = append(positional, a)
		}
	}
	if len(positional) > 0 {
		which = positional[0]
	}
	return which, dryRun, nil
}

func runTest(args []string, stdout, stderr io.Writer) int {
	which, dryRun, err := parseTestArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 64
	}

	kinds, err := testKinds(which)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 64
	}

	cfgPath, err := resolveConfigPath()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, "load config:", err)
		return 1
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	for i, k := range kinds {
		c := hook.SampleCard(k, cwd, cfg)
		if err := dispatchCard(c, dryRun, stdout); err != nil {
			fmt.Fprintln(stderr, "show:", err)
			return 1
		}
		if i < len(kinds)-1 {
			time.Sleep(400 * time.Millisecond)
		}
	}
	// Dry-run output must stay pure JSON so scripts (and tests) can parse it without a trailer to strip.
	if !isDryRun(dryRun) {
		fmt.Fprintf(stdout, "shown: %s sample(s)\n", which)
	}
	return 0
}

func testKinds(which string) ([]string, error) {
	switch which {
	case "ask", "permission", "idle", "done":
		return []string{which}, nil
	case "all":
		return []string{"ask", "permission", "idle", "done"}, nil
	default:
		return nil, fmt.Errorf("unknown test kind: %s (use ask, permission, idle, done or all)", which)
	}
}
