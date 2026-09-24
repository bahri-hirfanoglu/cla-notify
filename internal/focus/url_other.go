//go:build !windows

package focus

import "errors"

// RunURL backs the hidden "focus-url" subcommand; only the windows platform package implements it.
func RunURL(url string) error {
	return errors.New("focus-url is only supported on windows")
}
