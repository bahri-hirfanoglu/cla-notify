package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/bahri-hirfanoglu/cla-notify/internal/config"
	"github.com/bahri-hirfanoglu/cla-notify/internal/paths"
	"github.com/bahri-hirfanoglu/cla-notify/internal/present"
	"github.com/bahri-hirfanoglu/cla-notify/internal/terminal"
)

func runDoctor(stdout io.Writer) int {
	fmt.Fprintln(stdout, "cla-notify doctor")
	fmt.Fprintf(stdout, "  version: %s\n", Version)
	fmt.Fprintf(stdout, "  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	presenter := present.ForOS()
	fmt.Fprintf(stdout, "  presenter: %s\n", presenter.Name())
	if err := presenter.Check(); err != nil {
		fmt.Fprintf(stdout, "    check: %s\n", err)
	} else {
		fmt.Fprintln(stdout, "    check: ok")
	}

	cfgPath, err := resolveConfigPath()
	if err != nil {
		fmt.Fprintf(stdout, "  config path: error: %s\n", err)
	} else {
		exists := "missing"
		if _, statErr := os.Stat(cfgPath); statErr == nil {
			exists = "exists"
		}
		fmt.Fprintf(stdout, "  config path: %s (%s)\n", cfgPath, exists)
		raw, rawErr := config.LoadRaw(cfgPath)
		if rawErr != nil {
			fmt.Fprintf(stdout, "    validate: error: %s\n", rawErr)
		} else if problems := config.Validate(raw); len(problems) > 0 {
			fmt.Fprintln(stdout, "    validate: problems found")
			for _, p := range problems {
				fmt.Fprintf(stdout, "      %s\n", p)
			}
		} else {
			fmt.Fprintln(stdout, "    validate: ok")
		}
	}

	stateDir, err := paths.StateDir()
	if err != nil {
		fmt.Fprintf(stdout, "  state dir: error: %s\n", err)
	} else {
		fmt.Fprintf(stdout, "  state dir: %s\n", stateDir)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stdout, "  detected terminal: error: %s\n", err)
		return 0
	}
	focus := terminal.Detect(cwd, environMap())
	data, err := json.MarshalIndent(focus, "    ", "  ")
	if err == nil {
		fmt.Fprintln(stdout, "  detected terminal:")
		fmt.Fprintf(stdout, "    %s\n", string(data))
	}

	return 0
}

func environMap() map[string]string {
	env := os.Environ()
	m := make(map[string]string, len(env))
	for _, kv := range env {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
}
