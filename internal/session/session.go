// Package session persists per-session state (turn timing, model, terminal) between hook invocations.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
	"unicode"
)

// State is everything a hook invocation needs to remember about a running Claude Code session.
type State struct {
	TurnStartedAt *time.Time `json:"turnStartedAt,omitempty"`
	AskedAt       *time.Time `json:"askedAt,omitempty"`
	Model         string     `json:"model,omitempty"`
	LastDoneAt    *time.Time `json:"lastDoneAt,omitempty"`
}

// sanitize keeps only [A-Za-z0-9-_]: session_id comes from stdin, so it must never name a path outside sessions/.
func sanitize(id string) string {
	var b []rune
	for _, r := range id {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b = append(b, r)
		}
	}
	if len(b) == 0 {
		return "unknown"
	}
	return string(b)
}

func sessionPath(stateDir, sessionID string) string {
	return filepath.Join(stateDir, "sessions", sanitize(sessionID)+".json")
}

// Load reads the session's saved state, or a zero State when there is none yet.
func Load(stateDir, sessionID string) State {
	data, err := os.ReadFile(sessionPath(stateDir, sessionID))
	if err != nil {
		return State{}
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}
	}
	return s
}

// Save writes the session's state, creating the sessions directory as needed.
func Save(stateDir, sessionID string, s State) error {
	dir := filepath.Join(stateDir, "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath(stateDir, sessionID), data, 0o600)
}

// Cleanup is a best-effort GC: session files older than sessionMaxAge and cards older than cardMaxAge are dropped.
func Cleanup(stateDir string, sessionMaxAge, cardMaxAge time.Duration) {
	sweep(filepath.Join(stateDir, "sessions"), sessionMaxAge)
	sweep(filepath.Join(stateDir, "cards"), cardMaxAge)
}

func sweep(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
