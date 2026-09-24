package config

import (
	"fmt"
	"strconv"
	"strings"
)

// valueRule constrains what one schema path accepts, beyond its Go type; map keys are wildcarded as "*".
// This is the single table both README.md and commands/config.md describe.
type valueRule struct {
	enum         []string
	hasMin       bool
	min          float64
	minExclusive bool
	hasMax       bool
	max          float64
}

// eventNames are the fixed struct fields under "events": Events has no map, so each needs its own path.
var eventNames = []string{"ask", "permission", "idle", "done"}

var valueRules = buildValueRules()

// buildValueRules is the single source both README.md and commands/config.md describe by hand;
// it expands the per-event duration rules instead of repeating them four times.
func buildValueRules() map[string]valueRule {
	rules := map[string]valueRule{
		"display.corner":         {enum: []string{"top-right", "top-left", "bottom-right", "bottom-left"}},
		"display.screen":         {enum: []string{"auto", "main", "mouse"}},
		"display.theme":          {enum: []string{"auto", "dark", "light"}},
		"display.maxCards":       {hasMin: true, min: 1, hasMax: true, max: 10},
		"sound.volume":           {hasMin: true, min: 0, hasMax: true, max: 1},
		"contextWindow.default":  {hasMin: true, min: 0, minExclusive: true},
		"contextWindow.models.*": {hasMin: true, min: 0, minExclusive: true},
	}
	for _, name := range eventNames {
		rules["events."+name+".durationSeconds"] = valueRule{hasMin: true, min: 0}
		rules["events."+name+".minTurnSeconds"] = valueRule{hasMin: true, min: 0}
	}
	return rules
}

// checkValue reports why val fails the rule at canonicalPath, or "" when it passes or no rule applies.
func checkValue(canonicalPath string, val interface{}) string {
	rule, ok := valueRules[canonicalPath]
	if !ok {
		return ""
	}
	if rule.enum != nil {
		return checkEnum(rule, val)
	}
	return checkRange(rule, val)
}

func checkEnum(rule valueRule, val interface{}) string {
	s, ok := val.(string)
	if !ok {
		return ""
	}
	for _, allowed := range rule.enum {
		if s == allowed {
			return ""
		}
	}
	return "must be one of " + strings.Join(rule.enum, ", ")
}

func checkRange(rule valueRule, val interface{}) string {
	n, ok := val.(float64)
	if !ok {
		return ""
	}
	tooLow := rule.hasMin && (n < rule.min || (rule.minExclusive && n == rule.min))
	tooHigh := rule.hasMax && n > rule.max
	if !tooLow && !tooHigh {
		return ""
	}
	switch {
	case rule.hasMin && rule.hasMax:
		return fmt.Sprintf("must be between %s and %s", formatBound(rule.min), formatBound(rule.max))
	case rule.hasMin && rule.minExclusive:
		return fmt.Sprintf("must be greater than %s", formatBound(rule.min))
	case rule.hasMin:
		return fmt.Sprintf("must be at least %s", formatBound(rule.min))
	default:
		return fmt.Sprintf("must be at most %s", formatBound(rule.max))
	}
}

func formatBound(n float64) string {
	return strconv.FormatFloat(n, 'g', -1, 64)
}
