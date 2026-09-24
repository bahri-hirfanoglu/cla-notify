// Package config loads, merges, validates and edits the cla-notify config file.
package config

// Config is the fully merged (defaults + user overrides) configuration.
type Config struct {
	Enabled       bool          `json:"enabled"`
	Events        Events        `json:"events"`
	Display       Display       `json:"display"`
	Sound         Sound         `json:"sound"`
	Labels        Labels        `json:"labels"`
	ContextWindow ContextWindow `json:"contextWindow"`
}

// EventConfig is one entry under "events": ask, permission, idle or done.
type EventConfig struct {
	Enabled         bool    `json:"enabled"`
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	Sound           string  `json:"sound"`
	DurationSeconds float64 `json:"durationSeconds"`
	MinTurnSeconds  float64 `json:"minTurnSeconds,omitempty"`
}

// Events groups the four notification events.
type Events struct {
	Ask        EventConfig `json:"ask"`
	Permission EventConfig `json:"permission"`
	Idle       EventConfig `json:"idle"`
	Done       EventConfig `json:"done"`
}

// Display carries presentation settings passed straight through to the Card.
type Display struct {
	Corner              string `json:"corner"`
	Screen              string `json:"screen"`
	Theme               string `json:"theme"`
	RespectDock         bool   `json:"respectDock"`
	MaxCards            int    `json:"maxCards"`
	SuppressWhenFocused bool   `json:"suppressWhenFocused"`
	ShowOptions         bool   `json:"showOptions"`
	ShowContext         bool   `json:"showContext"`
	ShowModel           bool   `json:"showModel"`
	ShowGit             bool   `json:"showGit"`
}

// Sound holds the sound settings that are not per-event.
type Sound struct {
	Volume float64 `json:"volume"`
}

// Labels are the fixed words a presenter prints, taken from the config as templates.
type Labels struct {
	Ask           string `json:"ask"`
	Wait          string `json:"wait"`
	Done          string `json:"done"`
	FocusButton   string `json:"focusButton"`
	Dismiss       string `json:"dismiss"`
	MoreQuestions string `json:"moreQuestions"`
}

// ContextWindow resolves how large a model's context window is for the {context} placeholder.
type ContextWindow struct {
	Default int            `json:"default"`
	Models  map[string]int `json:"models"`
}

// Defaults returns the built-in configuration; every field here is the spec's documented default.
func Defaults() Config {
	return Config{
		Enabled: true,
		Events: Events{
			Ask:        EventConfig{Enabled: true, Title: "{project}: Claude has a question", Body: "{question}", Sound: "default", DurationSeconds: 20},
			Permission: EventConfig{Enabled: true, Title: "{project}: approval needed", Body: "{message}", Sound: "default", DurationSeconds: 20},
			Idle:       EventConfig{Enabled: true, Title: "{project}: waiting for your input", Body: "{message}", Sound: "default", DurationSeconds: 12},
			Done:       EventConfig{Enabled: true, Title: "{project}: done in {duration}", Body: "{summary}", Sound: "default", DurationSeconds: 8, MinTurnSeconds: 0},
		},
		Display: Display{
			Corner: "top-right", Screen: "auto", Theme: "auto", RespectDock: true, MaxCards: 3,
			SuppressWhenFocused: true, ShowOptions: true, ShowContext: true, ShowModel: true, ShowGit: true,
		},
		Sound: Sound{Volume: 0.6},
		Labels: Labels{
			Ask: "Question", Wait: "Waiting", Done: "Done",
			FocusButton: "Back to {app}", Dismiss: "Dismiss", MoreQuestions: "+{count} more questions",
		},
		ContextWindow: ContextWindow{
			Default: 200000,
			Models:  map[string]int{"opus": 1000000, "sonnet": 1000000},
		},
	}
}
