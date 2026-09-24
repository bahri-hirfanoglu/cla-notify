//go:build !linux

package present

import "errors"

// RunLinuxWorker backs the hidden "present-linux" subcommand; only the linux platform package implements it.
func RunLinuxWorker(path string) error {
	return errors.New("present-linux is only supported on linux")
}
