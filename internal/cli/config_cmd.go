package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bahri-hirfanoglu/cla-notify/internal/config"
)

func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cla-notify config <path|show|init|get|set|unset|validate|reset>")
		return 64
	}
	sub, rest := args[0], args[1:]

	cfgPath, err := resolveConfigPath()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	switch sub {
	case "path":
		fmt.Fprintln(stdout, cfgPath)
		return 0
	case "show":
		return runConfigShow(cfgPath, rest, stdout, stderr)
	case "init":
		return runConfigInit(cfgPath, rest, stdout, stderr)
	case "get":
		return runConfigGet(cfgPath, rest, stdout, stderr)
	case "set":
		return runConfigSet(cfgPath, rest, stderr)
	case "unset":
		return runConfigUnset(cfgPath, rest, stderr)
	case "validate":
		return runConfigValidate(cfgPath, stdout, stderr)
	case "reset":
		return runConfigReset(cfgPath, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown config subcommand: %s\n", sub)
		return 64
	}
}

func runConfigShow(cfgPath string, args []string, stdout, stderr io.Writer) int {
	showDefaults := false
	for _, a := range args {
		if a == "--defaults" {
			showDefaults = true
		}
	}

	var out interface{}
	if showDefaults {
		out = config.Defaults()
	} else {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		out = cfg
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func runConfigInit(cfgPath string, args []string, stdout, stderr io.Writer) int {
	force := false
	for _, a := range args {
		if a == "--force" {
			force = true
		}
	}
	if _, err := os.Stat(cfgPath); err == nil && !force {
		fmt.Fprintf(stderr, "config file already exists at %s (use --force to overwrite)\n", cfgPath)
		return 1
	}

	data, err := json.MarshalIndent(config.Defaults(), "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "wrote", cfgPath)
	return 0
}

func runConfigGet(cfgPath string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: cla-notify config get <key>")
		return 64
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	v, err := config.Get(cfg, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, config.FormatValue(v))
	return 0
}

func runConfigSet(cfgPath string, args []string, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: cla-notify config set <key> <value>")
		return 64
	}
	raw, err := config.LoadRaw(cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := config.Set(raw, args[0], args[1]); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := config.SaveRaw(cfgPath, raw); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runConfigUnset(cfgPath string, args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: cla-notify config unset <key>")
		return 64
	}
	raw, err := config.LoadRaw(cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := config.Unset(raw, args[0]); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := config.SaveRaw(cfgPath, raw); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runConfigValidate(cfgPath string, stdout, stderr io.Writer) int {
	raw, err := config.LoadRaw(cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	problems := config.Validate(raw)
	if len(problems) == 0 {
		fmt.Fprintln(stdout, "ok")
		return 0
	}
	for _, p := range problems {
		fmt.Fprintln(stdout, p)
	}
	return 1
}

func runConfigReset(cfgPath string, stdout, stderr io.Writer) int {
	if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "removed", cfgPath)
	return 0
}
