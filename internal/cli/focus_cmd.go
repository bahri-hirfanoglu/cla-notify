package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/focus"
	"github.com/bahri-hirfanoglu/cla-notify/internal/present"
)

func runFocus(args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: cla-notify focus <card.json>")
		return 64
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var c card.Card
	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := focus.Focus(c.Focus); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runVersion(stdout io.Writer) int {
	fmt.Fprintln(stdout, Version)
	return 0
}

func runPresentLinux(args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: cla-notify present-linux <card.json>")
		return 64
	}
	if err := present.RunLinuxWorker(args[0]); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runFocusURL(args []string, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: cla-notify focus-url <url>")
		return 64
	}
	if err := focus.RunURL(args[0]); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
