// Package hook parses a Claude Code hook event from stdin and turns it into a Card.
package hook

import "encoding/json"

// Input is the subset of the hook event JSON cla-notify reads.
type Input struct {
	SessionID        string          `json:"session_id"`
	TranscriptPath   string          `json:"transcript_path"`
	Cwd              string          `json:"cwd"`
	HookEventName    string          `json:"hook_event_name"`
	ToolName         string          `json:"tool_name"`
	ToolInput        json.RawMessage `json:"tool_input"`
	Message          string          `json:"message"`
	NotificationType string          `json:"notification_type"`
	Model            string          `json:"model"`
	StopHookActive   bool            `json:"stop_hook_active"`
}

type askUserQuestionInput struct {
	Questions []struct {
		Question string `json:"question"`
		Options  []struct {
			Label string `json:"label"`
		} `json:"options"`
	} `json:"questions"`
}
