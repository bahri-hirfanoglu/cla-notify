// Command cla-notify is the Claude Code notification plugin's Go entry point.
package main

import (
	"os"

	"github.com/bahri-hirfanoglu/cla-notify/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
