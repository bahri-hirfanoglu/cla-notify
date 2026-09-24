package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/config"
	"github.com/bahri-hirfanoglu/cla-notify/internal/session"
	"github.com/bahri-hirfanoglu/cla-notify/internal/transcript"
)

// askedAtGuard skips a Notification that arrives right after AskUserQuestion already showed a card.
const askedAtGuard = 10 * time.Second

// sessionMaxAge and cardMaxAge bound the best-effort state dir GC run on every hook call.
const (
	sessionMaxAge = 7 * 24 * time.Hour
	cardMaxAge    = time.Hour
)

// Deps carries what Process needs beyond the raw hook input; Now and Env are overridable for tests.
type Deps struct {
	ConfigPath string
	StateDir   string
	Now        func() time.Time
	Env        map[string]string
}

// Process handles one hook event and returns the Card to show, or nil when nothing should be shown.
// It never panics on malformed input; every error is returned for the caller to log, never thrown.
func Process(raw []byte, deps Deps) (*card.Card, error) {
	var in Input
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("parse hook input: %w", err)
	}
	if in.HookEventName == "" {
		return nil, nil
	}

	cfg, err := config.Load(deps.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !cfg.Enabled {
		return nil, nil
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}

	cwd := in.Cwd
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}

	sessionID := in.SessionID
	if sessionID == "" {
		sessionID = "unknown"
	}

	session.Cleanup(deps.StateDir, sessionMaxAge, cardMaxAge)

	st := session.Load(deps.StateDir, sessionID)
	if in.Model != "" {
		st.Model = in.Model
	}

	var result *card.Card
	switch in.HookEventName {
	case "SessionStart":
		// nothing to show; the model hint above is all SessionStart contributes.
	case "UserPromptSubmit":
		t := now()
		st.TurnStartedAt = &t
	case "PreToolUse":
		result = handleAskUserQuestion(in, cwd, sessionID, cfg, &st, now, deps.Env)
	case "Notification":
		result = handleNotification(in, cwd, sessionID, cfg, &st, now, deps.Env)
	case "Stop":
		result = handleStop(in, cwd, sessionID, cfg, &st, now, deps.Env)
	}

	if err := session.Save(deps.StateDir, sessionID, st); err != nil {
		return result, fmt.Errorf("save session: %w", err)
	}
	return result, nil
}

func handleAskUserQuestion(in Input, cwd, sessionID string, cfg config.Config, st *session.State, now func() time.Time, env map[string]string) *card.Card {
	if in.ToolName != "AskUserQuestion" || !cfg.Events.Ask.Enabled || len(in.ToolInput) == 0 {
		return nil
	}
	var toolInput askUserQuestionInput
	if err := json.Unmarshal(in.ToolInput, &toolInput); err != nil || len(toolInput.Questions) == 0 {
		return nil
	}
	first := toolInput.Questions[0]
	var options []string
	for _, o := range first.Options {
		options = append(options, o.Label)
	}

	t := now()
	st.AskedAt = &t

	return buildCard(cardParams{
		kind: card.KindAsk, event: "ask", sessionID: sessionID, cwd: cwd,
		eventCfg: cfg.Events.Ask, cfg: cfg, state: *st, now: now(), env: env,
		question: first.Question, options: options, questionCount: len(toolInput.Questions),
		summary: transcript.Summarize(in.TranscriptPath),
	})
}

func handleNotification(in Input, cwd, sessionID string, cfg config.Config, st *session.State, now func() time.Time, env map[string]string) *card.Card {
	if st.AskedAt != nil && now().Sub(*st.AskedAt) < askedAtGuard {
		return nil
	}

	var event string
	switch in.NotificationType {
	case "idle_prompt":
		event = "idle"
	case "", "agent_needs_input", "permission_prompt", "elicitation_dialog", "elicitation_url_dialog":
		event = "permission"
	default:
		return nil
	}

	eventCfg := cfg.Events.Permission
	if event == "idle" {
		eventCfg = cfg.Events.Idle
	}
	if !eventCfg.Enabled {
		return nil
	}

	return buildCard(cardParams{
		kind: card.KindWait, event: event, sessionID: sessionID, cwd: cwd,
		eventCfg: eventCfg, cfg: cfg, state: *st, now: now(), env: env,
		message: in.Message,
		summary: transcript.Summarize(in.TranscriptPath),
	})
}

func handleStop(in Input, cwd, sessionID string, cfg config.Config, st *session.State, now func() time.Time, env map[string]string) *card.Card {
	if in.StopHookActive {
		return nil
	}
	if !cfg.Events.Done.Enabled {
		return nil
	}

	summary := transcript.Summarize(in.TranscriptPath)

	var elapsed *float64
	switch {
	case st.TurnStartedAt != nil:
		e := now().Sub(*st.TurnStartedAt).Seconds()
		elapsed = &e
	case !summary.LastUserTimestamp.IsZero():
		e := now().Sub(summary.LastUserTimestamp).Seconds()
		elapsed = &e
	}

	nowTime := now()
	st.TurnStartedAt = nil
	st.AskedAt = nil
	st.LastDoneAt = &nowTime

	if elapsed != nil && *elapsed < cfg.Events.Done.MinTurnSeconds {
		return nil
	}

	return buildCard(cardParams{
		kind: card.KindDone, event: "done", sessionID: sessionID, cwd: cwd,
		eventCfg: cfg.Events.Done, cfg: cfg, state: *st, now: nowTime, env: env,
		summary: summary, elapsedSeconds: elapsed,
	})
}
