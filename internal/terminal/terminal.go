// Package terminal detects the terminal, tab and pane the Claude session runs in.
package terminal

import "github.com/bahri-hirfanoglu/cla-notify/internal/card"

// Detect fills a FocusTarget from the environment, then lets the platform add its own details.
func Detect(cwd string, env map[string]string) card.FocusTarget {
	t := detectCommon(cwd, env)
	addPlatformDetails(&t, env)
	return t
}

// addPlatformDetails is replaced per platform in init; the default adds nothing.
var addPlatformDetails = func(*card.FocusTarget, map[string]string) {}
