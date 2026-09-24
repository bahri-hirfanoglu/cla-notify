package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// logCap and logTrimTo cap the hook's error log so a long-running machine never grows it unbounded.
const (
	logCap    = 1 << 20 // 1 MiB
	logTrimTo = 512 * 1024
)

// logError best-effort appends a timestamped line to <stateDir>/cla-notify.log; failures are swallowed,
// because the hook must never fail loudly.
func logError(stateDir string, err error) {
	if stateDir == "" || err == nil {
		return
	}
	if mkErr := os.MkdirAll(stateDir, 0o700); mkErr != nil {
		return
	}
	path := filepath.Join(stateDir, "cla-notify.log")
	rotateLogIfNeeded(path)

	f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if openErr != nil {
		return
	}
	defer f.Close()
	line := fmt.Sprintf("%s %s\n", time.Now().UTC().Format(time.RFC3339), err.Error())
	_, _ = f.WriteString(line)
}

func rotateLogIfNeeded(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < logCap {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if len(data) > logTrimTo {
		data = data[len(data)-logTrimTo:]
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		}
	}
	_ = os.WriteFile(path, data, 0o600)
}
