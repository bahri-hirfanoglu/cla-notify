//go:build !darwin && !linux && !windows

package terminal

import "github.com/bahri-hirfanoglu/cla-notify/internal/card"

func init() {
	addPlatformDetails = func(*card.FocusTarget, map[string]string) {}
}
