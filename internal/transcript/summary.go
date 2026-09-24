package transcript

import (
	"regexp"
	"strings"
)

// FirstProseSentence extracts the first prose sentence of text: the first non-code, non-table,
// non-heading-marker line, markdown stripped, cut at the first sentence boundary, max 200 chars.
func FirstProseSentence(text string) string {
	line := firstProseLine(text)
	if line == "" {
		return ""
	}
	clean := stripMarkdown(line)
	if clean == "" {
		return ""
	}
	return truncate(firstSentence(clean), 200)
}

func firstProseLine(text string) string {
	inFence := false
	for _, rawLine := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, ">") {
			continue
		}
		if isHorizontalRule(trimmed) {
			continue
		}
		if stripped := stripLeadingHeading(trimmed); stripped != "" {
			return stripped
		}
	}
	return ""
}

func isHorizontalRule(s string) bool {
	if s == "" || len(s) < 3 {
		return false
	}
	first := s[0]
	if first != '-' && first != '*' && first != '_' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

func stripLeadingHeading(s string) string {
	count := 0
	i := 0
	for i < len(s) && s[i] == '#' && count < 6 {
		i++
		count++
	}
	return strings.TrimSpace(s[i:])
}

var (
	reLink       = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	reCode       = regexp.MustCompile("`+([^`]*)`+")
	reBold       = regexp.MustCompile(`\*\*([^*]*)\*\*`)
	reBoldU      = regexp.MustCompile(`__([^_]*)__`)
	reItalic     = regexp.MustCompile(`\*([^*]+)\*`)
	reStrike     = regexp.MustCompile(`~~([^~]*)~~`)
	reHeading    = regexp.MustCompile(`(?m)^[ \t]*#{1,6}[ \t]*`)
	reBullet     = regexp.MustCompile(`(?m)^[ \t]*[-*+][ \t]+`)
	reBlockquote = regexp.MustCompile(`(?m)^[ \t]*>[ \t]?`)
	reSpaces     = regexp.MustCompile(` +`)
)

// stripMarkdown strips inline and block markdown syntax to prose, mirroring common formatting in assistant answers.
func stripMarkdown(s string) string {
	s = reLink.ReplaceAllString(s, "$1")
	s = reCode.ReplaceAllString(s, "$1")
	s = reBold.ReplaceAllString(s, "$1")
	s = reBoldU.ReplaceAllString(s, "$1")
	s = reItalic.ReplaceAllString(s, "$1")
	s = reStrike.ReplaceAllString(s, "$1")
	s = reHeading.ReplaceAllString(s, "")
	s = reBullet.ReplaceAllString(s, "")
	s = reBlockquote.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "|", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// firstSentence cuts s at the first ".", "!" or "?" followed by a space or the end of the string.
func firstSentence(s string) string {
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' {
			rest := s[i+1:]
			if rest == "" || strings.HasPrefix(rest, " ") {
				return strings.TrimSpace(s[:i+1])
			}
		}
	}
	return s
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
