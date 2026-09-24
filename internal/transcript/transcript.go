// Package transcript reads a Claude Code JSONL transcript for the last answer, model and context usage.
package transcript

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

// tailWindow bounds how much of a long transcript is read; headWindow is the fallback scan for the model attachment.
const (
	tailWindow = 1_000_000
	headWindow = 512_000
)

// Summary is everything the hook needs from a transcript to fill in model, context and elapsed time.
type Summary struct {
	LastAssistantText  string
	LastAssistantModel string
	UsedTokens         *int
	LastUserTimestamp  time.Time
	// SessionModelID is Claude Code's own model id, e.g. "claude-opus-5-5[1m]"; the only place the 1M hint appears.
	SessionModelID string
}

// Summarize reads only the tail of path so a long-running session never costs more than a bounded seek and parse.
func Summarize(path string) Summary {
	var summary Summary
	if path == "" {
		return summary
	}
	f, err := os.Open(path)
	if err != nil {
		return summary
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return summary
	}
	size := info.Size()
	start := int64(0)
	if size > tailWindow {
		start = size - tailWindow
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return summary
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return summary
	}

	lines := strings.Split(string(data), "\n")
	var lastAssistantMessage map[string]interface{}

	for i, rawLine := range lines {
		// A mid-file byte seek can land inside a line; that fragment never parses, so skip it.
		if start > 0 && i == 0 {
			continue
		}
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if sidechain, ok := obj["isSidechain"].(bool); ok && sidechain {
			continue
		}
		if model := modelIdentity(obj); model != "" {
			summary.SessionModelID = model
			continue
		}
		typ, _ := obj["type"].(string)
		message, hasMessage := obj["message"].(map[string]interface{})
		if !hasMessage {
			continue
		}

		switch typ {
		case "assistant":
			lastAssistantMessage = message
		case "user":
			if isRealUserPrompt(message) {
				if ts, ok := obj["timestamp"].(string); ok {
					if t, err := parseTimestamp(ts); err == nil {
						summary.LastUserTimestamp = t
					}
				}
			}
		}
	}

	// The model attachment is written at session start, so a long transcript only has it near the head.
	if summary.SessionModelID == "" && start > 0 {
		if _, err := f.Seek(0, io.SeekStart); err == nil {
			head := make([]byte, headWindow)
			n, _ := f.Read(head)
			summary.SessionModelID = headModelIdentity(head[:n])
		}
	}

	if lastAssistantMessage != nil {
		if m, ok := lastAssistantMessage["model"].(string); ok {
			summary.LastAssistantModel = m
		}
		if content, ok := lastAssistantMessage["content"].([]interface{}); ok {
			for i := len(content) - 1; i >= 0; i-- {
				block, ok := content[i].(map[string]interface{})
				if !ok {
					continue
				}
				if t, _ := block["type"].(string); t == "text" {
					if txt, ok := block["text"].(string); ok {
						summary.LastAssistantText = txt
					}
					break
				}
			}
		}
		if usage, ok := lastAssistantMessage["usage"].(map[string]interface{}); ok {
			used := intValue(usage["input_tokens"]) + intValue(usage["cache_read_input_tokens"]) + intValue(usage["cache_creation_input_tokens"])
			summary.UsedTokens = &used
		}
	}

	return summary
}

func modelIdentity(obj map[string]interface{}) string {
	if t, _ := obj["type"].(string); t != "attachment" {
		return ""
	}
	attachment, ok := obj["attachment"].(map[string]interface{})
	if !ok {
		return ""
	}
	if at, _ := attachment["type"].(string); at != "model" {
		return ""
	}
	identity, ok := attachment["identity"].(map[string]interface{})
	if !ok {
		return ""
	}
	id, _ := identity["modelId"].(string)
	return id
}

// headModelIdentity keeps the last modelId match in data, matching how the tail scan keeps the latest one seen.
func headModelIdentity(data []byte) string {
	var found string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "\"modelId\"") {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if m := modelIdentity(obj); m != "" {
			found = m
		}
	}
	return found
}

// isRealUserPrompt reports whether message has text content; a turn that is only a tool_result is not one.
func isRealUserPrompt(message map[string]interface{}) bool {
	if text, ok := message["content"].(string); ok {
		return text != ""
	}
	if blocks, ok := message["content"].([]interface{}); ok {
		for _, b := range blocks {
			block, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			if t, _ := block["type"].(string); t == "text" {
				return true
			}
		}
	}
	return false
}

func intValue(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
