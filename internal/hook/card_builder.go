package hook

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/config"
	"github.com/bahri-hirfanoglu/cla-notify/internal/gitinfo"
	"github.com/bahri-hirfanoglu/cla-notify/internal/paths"
	"github.com/bahri-hirfanoglu/cla-notify/internal/render"
	"github.com/bahri-hirfanoglu/cla-notify/internal/session"
	"github.com/bahri-hirfanoglu/cla-notify/internal/terminal"
	"github.com/bahri-hirfanoglu/cla-notify/internal/transcript"
)

type cardParams struct {
	kind           card.Kind
	event          string
	sessionID      string
	cwd            string
	eventCfg       config.EventConfig
	cfg            config.Config
	state          session.State
	now            time.Time
	env            map[string]string
	question       string
	options        []string
	questionCount  int
	message        string
	summary        transcript.Summary
	elapsedSeconds *float64
}

func buildCard(p cardParams) *card.Card {
	project := resolveProject(p.cwd, p.cfg.Display.ShowGit)

	var modelDisplay string
	if p.cfg.Display.ShowModel {
		raw := p.summary.LastAssistantModel
		if raw == "" {
			raw = p.state.Model
		}
		if raw != "" {
			modelDisplay = transcript.DisplayName(raw)
		}
	}

	var ctx *card.Context
	var contextText, contextPercentText string
	if p.cfg.Display.ShowContext && p.summary.UsedTokens != nil {
		used := *p.summary.UsedTokens
		window := resolveContextWindow(p.cfg.ContextWindow, p.summary.SessionModelID, p.summary.LastAssistantModel, used)
		ctx = &card.Context{UsedTokens: used, WindowTokens: window}
		contextText = render.FormatContext(used, window)
		contextPercentText = render.FormatContextPercent(used, window)
	}

	env := p.env
	if env == nil {
		env = environMap()
	}
	focus := terminal.Detect(p.cwd, env)

	options := p.options
	if !p.cfg.Display.ShowOptions {
		options = nil
	}

	// Presenters decode options as an array, so never emit null.
	if options == nil {
		options = []string{}
	}

	var durationText string
	if p.elapsedSeconds != nil {
		durationText = render.FormatDuration(*p.elapsedSeconds)
	}

	vars := render.Vars{
		Project: project.Name, Path: project.Path, Branch: project.Branch, Repo: project.Remote,
		Model: modelDisplay, Context: contextText, ContextPercent: contextPercentText,
		Duration: durationText, Question: p.question, Options: strings.Join(options, ", "),
		Message: p.message, Summary: transcript.FirstProseSentence(p.summary.LastAssistantText),
		App: focus.AppName, Event: p.event,
	}

	title := render.Render(p.eventCfg.Title, vars)
	body := render.Render(p.eventCfg.Body, vars)

	var labelKind string
	switch p.kind {
	case card.KindAsk:
		labelKind = p.cfg.Labels.Ask
	case card.KindWait:
		labelKind = p.cfg.Labels.Wait
	default:
		labelKind = p.cfg.Labels.Done
	}

	// moreQuestions counts every question beyond the one already shown as the card's title/body.
	var moreQuestions string
	if p.questionCount > 1 {
		countVars := vars
		countVars.Count = strconv.Itoa(p.questionCount - 1)
		moreQuestions = render.Render(p.cfg.Labels.MoreQuestions, countVars)
	}

	return &card.Card{
		Version:        card.Version,
		Kind:           p.kind,
		Event:          p.event,
		SessionID:      p.sessionID,
		Title:          title,
		Body:           body,
		Options:        options,
		QuestionCount:  p.questionCount,
		Project:        project,
		Model:          modelDisplay,
		Context:        ctx,
		ElapsedSeconds: p.elapsedSeconds,
		Focus:          focus,
		Sound:          resolveSound(p.eventCfg.Sound),
		SoundVolume:    p.cfg.Sound.Volume,
		DurationSecs:   p.eventCfg.DurationSeconds,
		CreatedAt:      p.now,
		Display: card.Display{
			Corner: p.cfg.Display.Corner, Screen: p.cfg.Display.Screen, Theme: p.cfg.Display.Theme,
			RespectDock: p.cfg.Display.RespectDock, MaxCards: p.cfg.Display.MaxCards,
			SuppressWhenFocused: p.cfg.Display.SuppressWhenFocused,
			ShowOptions:         p.cfg.Display.ShowOptions, ShowContext: p.cfg.Display.ShowContext,
			ShowModel: p.cfg.Display.ShowModel, ShowGit: p.cfg.Display.ShowGit,
		},
		Labels: card.Labels{
			Kind:          render.Render(labelKind, vars),
			FocusButton:   render.Render(p.cfg.Labels.FocusButton, vars),
			Dismiss:       render.Render(p.cfg.Labels.Dismiss, vars),
			Context:       contextText,
			Elapsed:       durationText,
			MoreQuestions: moreQuestions,
		},
	}
}

func resolveProject(cwd string, showGit bool) card.Project {
	if !showGit {
		return card.Project{Name: filepath.Base(cwd), Path: cwd}
	}
	gi := gitinfo.Lookup(cwd)
	return card.Project{Name: gi.Name, Path: gi.Path, Branch: gi.Branch, Remote: gi.Remote, Dirty: gi.Dirty}
}

// resolveContextWindow prefers a model-family override from config, then Claude Code's own "[1m]" hint,
// then a large-usage heuristic, then the configured default.
func resolveContextWindow(cw config.ContextWindow, sessionModel, assistantModel string, used int) int {
	family := modelFamily(assistantModel)
	if family == "" {
		family = modelFamily(sessionModel)
	}
	if family != "" {
		for name, window := range cw.Models {
			if strings.EqualFold(name, family) {
				return window
			}
		}
	}
	if strings.Contains(sessionModel, "[1m]") {
		return 1_000_000
	}
	if used > 200_000 {
		return 1_000_000
	}
	if cw.Default > 0 {
		return cw.Default
	}
	return 200_000
}

func modelFamily(raw string) string {
	id := strings.TrimPrefix(raw, "claude-")
	if idx := strings.IndexByte(id, '-'); idx >= 0 {
		id = id[:idx]
	} else if idx := strings.IndexByte(id, '['); idx >= 0 {
		id = id[:idx]
	}
	return strings.ToLower(id)
}

// resolveSound expands "~" in a sound file path and passes "default"/"" through for the presenter to interpret.
func resolveSound(raw string) string {
	switch strings.ToLower(raw) {
	case "none", "":
		return ""
	case "default":
		return "default"
	default:
		return paths.Expand(raw)
	}
}

func environMap() map[string]string {
	env := os.Environ()
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			m[kv[:idx]] = kv[idx+1:]
		}
	}
	return m
}
