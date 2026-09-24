package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeBasicUsage(t *testing.T) {
	s := Summarize("testdata/basic.jsonl")
	if s.SessionModelID != "claude-opus-5-5[1m]" {
		t.Errorf("SessionModelID = %q, want claude-opus-5-5[1m]", s.SessionModelID)
	}
	if s.LastAssistantModel != "claude-opus-5-5" {
		t.Errorf("LastAssistantModel = %q, want claude-opus-5-5", s.LastAssistantModel)
	}
	if s.LastAssistantText != "I fixed the bug and ran the tests." {
		t.Errorf("LastAssistantText = %q", s.LastAssistantText)
	}
	if s.UsedTokens == nil || *s.UsedTokens != 3500 {
		t.Errorf("UsedTokens = %v, want 3500 (1000+2000+500)", s.UsedTokens)
	}
	if s.LastUserTimestamp.IsZero() {
		t.Error("LastUserTimestamp should be set from the real user prompt")
	}
}

func TestSummarizeMarkdownSummary(t *testing.T) {
	s := Summarize("testdata/markdown.jsonl")
	got := FirstProseSentence(s.LastAssistantText)
	want := "Here is bold and code and a link."
	if got != want {
		t.Errorf("FirstProseSentence() = %q, want %q", got, want)
	}
}

func TestSummarizeSkipsSidechain(t *testing.T) {
	s := Summarize("testdata/sidechain.jsonl")
	if s.LastAssistantModel != "claude-sonnet-5" {
		t.Errorf("LastAssistantModel = %q, want claude-sonnet-5 (sidechain line skipped)", s.LastAssistantModel)
	}
	if s.UsedTokens == nil || *s.UsedTokens != 100 {
		t.Errorf("UsedTokens = %v, want 100", s.UsedTokens)
	}
}

func TestSummarizeMultiBlockPicksLastTextBlock(t *testing.T) {
	s := Summarize("testdata/multiblock.jsonl")
	if s.LastAssistantText != "second block wins" {
		t.Errorf("LastAssistantText = %q, want last text block", s.LastAssistantText)
	}
}

func TestSummarizeIgnoresToolResultOnlyUserTurn(t *testing.T) {
	s := Summarize("testdata/toolresult.jsonl")
	if !s.LastUserTimestamp.IsZero() {
		t.Error("a tool_result-only user turn should not set LastUserTimestamp")
	}
}

func TestSummarizeMissingFile(t *testing.T) {
	s := Summarize(filepath.Join(t.TempDir(), "missing.jsonl"))
	if s.LastAssistantText != "" || s.UsedTokens != nil {
		t.Errorf("Summarize() of a missing file = %+v, want zero value", s)
	}
}

func TestSummarizeEmptyPath(t *testing.T) {
	s := Summarize("")
	if s.LastAssistantText != "" {
		t.Errorf("Summarize(\"\") = %+v, want zero value", s)
	}
}

// TestSummarizeLargeFileHeadFallback checks that a model attachment near the start of a transcript
// larger than the tail window is still found through the head-scan fallback.
func TestSummarizeLargeFileHeadFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fmt.Fprintln(f, `{"type":"attachment","attachment":{"type":"model","identity":{"modelId":"claude-opus-5-5[1m]"}}}`)
	filler := `{"type":"other","padding":"` + strings.Repeat("x", 200) + `"}`
	for i := 0; i < 6000; i++ {
		fmt.Fprintln(f, filler)
	}
	fmt.Fprintln(f, `{"type":"assistant","message":{"model":"claude-opus-5-5","content":[{"type":"text","text":"final answer"}]}}`)

	s := Summarize(path)
	if s.SessionModelID != "claude-opus-5-5[1m]" {
		t.Errorf("SessionModelID = %q, want claude-opus-5-5[1m] via head fallback", s.SessionModelID)
	}
	if s.LastAssistantText != "final answer" {
		t.Errorf("LastAssistantText = %q, want final answer", s.LastAssistantText)
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"claude-opus-5-5", "Opus 5.5"},
		{"claude-sonnet-5", "Sonnet 5"},
		{"claude-opus-5-5-20250101", "Opus 5.5"},
		{"claude-haiku-3-5-20241022", "Haiku 3.5"},
		{"opus-5-5", "Opus 5.5"},
		{"", ""},
		{"-", "-"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := DisplayName(tt.raw); got != tt.want {
				t.Errorf("DisplayName(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestFirstProseSentence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text keeps only the first sentence",
			in:   "First sentence. Second sentence.",
			want: "First sentence.",
		},
		{
			name: "skips a fenced code block",
			in:   "```go\nfmt.Println(1)\n```\nActual prose here.",
			want: "Actual prose here.",
		},
		{
			name: "skips a table row",
			in:   "| a | b |\n|---|---|\nReal text.",
			want: "Real text.",
		},
		{
			name: "skips a blockquote",
			in:   "> quoted aside\nReal text.",
			want: "Real text.",
		},
		{
			name: "skips a horizontal rule",
			in:   "---\nReal text.",
			want: "Real text.",
		},
		{
			name: "strips a leading heading marker",
			in:   "## Summary\nmore text",
			want: "Summary",
		},
		{
			name: "no sentence terminator returns the whole line",
			in:   "no terminator here",
			want: "no terminator here",
		},
		{
			name: "empty text",
			in:   "",
			want: "",
		},
		{
			name: "only blank lines",
			in:   "\n\n   \n",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FirstProseSentence(tt.in); got != tt.want {
				t.Errorf("FirstProseSentence(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFirstProseSentenceTruncatesAt200Chars(t *testing.T) {
	long := ""
	for i := 0; i < 50; i++ {
		long += "0123456789"
	}
	got := FirstProseSentence(long)
	if len(got) != 200 {
		t.Fatalf("len(FirstProseSentence()) = %d, want 200", len(got))
	}
	if got[197:] != "..." {
		t.Errorf("FirstProseSentence() = %q, want to end with ...", got)
	}
}
