package hook

import (
	"time"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/config"
	"github.com/bahri-hirfanoglu/cla-notify/internal/transcript"
)

// SampleCard builds a representative Card for `cla-notify test`, using the real config and terminal detection.
func SampleCard(event, cwd string, cfg config.Config) *card.Card {
	now := time.Now()
	switch event {
	case "ask":
		return buildCard(cardParams{
			kind: card.KindAsk, event: "ask", sessionID: "test-ask", cwd: cwd,
			eventCfg: cfg.Events.Ask, cfg: cfg, now: now,
			question:      "Which approach do you prefer: keep the current structure or refactor it?",
			options:       []string{"Keep current structure", "Refactor", "Try both"},
			questionCount: 1,
			summary:       sampleSummary(),
		})
	case "permission":
		return buildCard(cardParams{
			kind: card.KindWait, event: "permission", sessionID: "test-permission", cwd: cwd,
			eventCfg: cfg.Events.Permission, cfg: cfg, now: now,
			message: "Claude is waiting for approval to delete a file.",
			summary: sampleSummary(),
		})
	case "idle":
		return buildCard(cardParams{
			kind: card.KindWait, event: "idle", sessionID: "test-idle", cwd: cwd,
			eventCfg: cfg.Events.Idle, cfg: cfg, now: now,
			message: "Claude is waiting for your input.",
			summary: sampleSummary(),
		})
	case "done":
		elapsed := 134.0
		return buildCard(cardParams{
			kind: card.KindDone, event: "done", sessionID: "test-done", cwd: cwd,
			eventCfg: cfg.Events.Done, cfg: cfg, now: now,
			summary: sampleSummary(), elapsedSeconds: &elapsed,
		})
	default:
		return nil
	}
}

func sampleSummary() transcript.Summary {
	used := 84000
	return transcript.Summary{
		LastAssistantText:  "Applied the changes and ran the tests, all green.",
		LastAssistantModel: "claude-opus-5-5",
		UsedTokens:         &used,
	}
}
