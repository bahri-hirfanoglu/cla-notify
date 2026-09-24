package cli

import (
	"fmt"
	"io"

	"github.com/bahri-hirfanoglu/cla-notify/internal/hook"
	"github.com/bahri-hirfanoglu/cla-notify/internal/paths"
)

// runHook always returns 0: the hook must never fail loudly, so every error is logged instead of surfaced.
func runHook(stdin io.Reader, stdout io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			if stateDir, err := resolveStateDirQuiet(); err == nil {
				logError(stateDir, fmt.Errorf("panic: %v", r))
			}
		}
		code = 0
	}()

	data, err := io.ReadAll(stdin)
	if err != nil {
		return 0
	}

	stateDir, err := resolveStateDirQuiet()
	if err != nil {
		return 0
	}

	cfgPath, err := resolveConfigPath()
	if err != nil {
		logError(stateDir, err)
		return 0
	}

	c, err := hook.Process(data, hook.Deps{ConfigPath: cfgPath, StateDir: stateDir})
	if err != nil {
		logError(stateDir, err)
		return 0
	}
	if c == nil {
		return 0
	}

	if err := dispatchCard(c, false, stdout); err != nil {
		logError(stateDir, err)
	}
	return 0
}

func resolveStateDirQuiet() (string, error) {
	return paths.StateDir()
}
