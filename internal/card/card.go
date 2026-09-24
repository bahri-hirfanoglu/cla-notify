// Package card defines the JSON document every presenter receives.
package card

import "time"

// Version is bumped whenever a field is added, removed or changes meaning.
const Version = 1

// Kind drives the accent colour, glyph and sound of a card.
type Kind string

const (
	KindAsk  Kind = "ask"  // Claude asked a question (AskUserQuestion)
	KindWait Kind = "wait" // a permission prompt or an idle input wait
	KindDone Kind = "done" // the turn finished
)

// Card is a fully rendered notification: presenters draw it and never read the config.
type Card struct {
	Version        int         `json:"version"`
	Kind           Kind        `json:"kind"`
	Event          string      `json:"event"` // config event name: ask, permission, idle or done
	SessionID      string      `json:"sessionId"`
	Title          string      `json:"title"`
	Body           string      `json:"body"`
	Options        []string    `json:"options"`
	QuestionCount  int         `json:"questionCount"`
	Project        Project     `json:"project"`
	Model          string      `json:"model,omitempty"`
	Context        *Context    `json:"context,omitempty"`
	ElapsedSeconds *float64    `json:"elapsedSeconds,omitempty"`
	Focus          FocusTarget `json:"focus"`
	Sound          string      `json:"sound,omitempty"` // resolved sound file path or platform sound name; empty is silent
	SoundVolume    float64     `json:"soundVolume"`
	DurationSecs   float64     `json:"durationSeconds"`
	CreatedAt      time.Time   `json:"createdAt"`
	Display        Display     `json:"display"`
	Labels         Labels      `json:"labels"`
}

// Project describes where the session runs.
type Project struct {
	Name   string `json:"name"`             // directory basename
	Path   string `json:"path"`             // absolute cwd
	Branch string `json:"branch,omitempty"` // empty outside a git repo or on a detached HEAD
	Remote string `json:"remote,omitempty"` // "owner/repo" or host path, credentials stripped
	Dirty  *bool  `json:"dirty,omitempty"`  // uncommitted changes present
}

// Context is the session's context window usage.
type Context struct {
	UsedTokens   int `json:"usedTokens"`
	WindowTokens int `json:"windowTokens"`
}

// FocusTarget holds what each platform needs to bring the originating terminal back.
type FocusTarget struct {
	Terminal       string `json:"terminal"` // TERM_PROGRAM or detected app, e.g. "iTerm.app", "vscode", "WindowsTerminal"
	AppName        string `json:"appName"`  // human name for the focus button, e.g. "iTerm2"
	BundleID       string `json:"bundleId,omitempty"`
	ItermSessionID string `json:"itermSessionId,omitempty"`
	TTY            string `json:"tty,omitempty"`
	TmuxPane       string `json:"tmuxPane,omitempty"`
	TmuxSocket     string `json:"tmuxSocket,omitempty"`
	WeztermPane    string `json:"weztermPane,omitempty"`
	KittyWindowID  string `json:"kittyWindowId,omitempty"`
	KittyListenOn  string `json:"kittyListenOn,omitempty"`
	X11WindowID    string `json:"x11WindowId,omitempty"` // linux: WINDOWID of the terminal, when set
	WindowsHWND    string `json:"windowsHwnd,omitempty"` // windows: top-level window handle of the terminal
	TerminalPID    int    `json:"terminalPid,omitempty"` // pid of the terminal application
	WTSession      string `json:"wtSession,omitempty"`   // windows: WT_SESSION of Windows Terminal
	CWD            string `json:"cwd"`
}

// Display carries the presentation settings from the config.
type Display struct {
	Corner              string `json:"corner"` // top-right, top-left, bottom-right, bottom-left
	Screen              string `json:"screen"` // auto, main or mouse
	Theme               string `json:"theme"`  // auto, dark or light
	RespectDock         bool   `json:"respectDock"`
	MaxCards            int    `json:"maxCards"`
	SuppressWhenFocused bool   `json:"suppressWhenFocused"`
	ShowOptions         bool   `json:"showOptions"`
	ShowContext         bool   `json:"showContext"`
	ShowModel           bool   `json:"showModel"`
	ShowGit             bool   `json:"showGit"`
}

// Labels are the fixed words a presenter prints, already rendered from the config.
type Labels struct {
	Kind          string `json:"kind"`          // e.g. "Question", "Waiting", "Done"
	FocusButton   string `json:"focusButton"`   // e.g. "Back to iTerm2"
	Dismiss       string `json:"dismiss"`       // e.g. "Dismiss"
	Context       string `json:"context"`       // e.g. "84k / 1M"
	Elapsed       string `json:"elapsed"`       // e.g. "2m 14s"
	MoreQuestions string `json:"moreQuestions"` // e.g. "+2 more questions"; empty when questionCount <= 1
}
