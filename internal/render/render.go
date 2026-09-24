// Package render turns a title/body template plus known values into final text.
package render

import (
	"regexp"
	"strings"
)

// Vars holds every placeholder value a template may reference; a zero field renders as empty.
type Vars struct {
	Project        string
	Path           string
	Branch         string
	Repo           string
	Model          string
	Context        string
	ContextPercent string
	Duration       string
	Question       string
	Options        string
	Message        string
	Summary        string
	App            string
	Event          string
	Count          string
}

var placeholderPattern = regexp.MustCompile(`\{[a-zA-Z]+\}`)

// Render substitutes every known {placeholder} in tmpl and trims dangling trailing separators.
// A placeholder not in Vars is left as written; Count is only set by the caller rendering labels.moreQuestions.
func Render(tmpl string, v Vars) string {
	known := map[string]string{
		"project": v.Project, "path": v.Path, "branch": v.Branch, "repo": v.Repo,
		"model": v.Model, "context": v.Context, "contextPercent": v.ContextPercent,
		"duration": v.Duration, "question": v.Question, "options": v.Options,
		"message": v.Message, "summary": v.Summary, "app": v.App, "event": v.Event,
		"count": v.Count,
	}
	result := placeholderPattern.ReplaceAllStringFunc(tmpl, func(token string) string {
		name := token[1 : len(token)-1]
		if val, ok := known[name]; ok {
			return val
		}
		return token
	})
	return trimTrailingSeparators(result)
}

// trimTrailingSeparators drops a dangling separator (and the whitespace around it) left by an empty placeholder.
func trimTrailingSeparators(s string) string {
	for {
		next := strings.TrimRight(s, " \t")
		next = strings.TrimRight(next, ":,-")
		next = strings.TrimRight(next, " \t")
		if next == s {
			return next
		}
		s = next
	}
}
