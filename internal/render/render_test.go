package render

import "testing"

func TestRenderPlaceholders(t *testing.T) {
	v := Vars{
		Project: "myproj", Path: "/repo", Branch: "main", Repo: "owner/myproj",
		Model: "Opus 5.5", Context: "84k / 1M", ContextPercent: "8%",
		Duration: "2m 14s", Question: "Pick one?", Options: "A, B",
		Message: "waiting", Summary: "Applied the fix.", App: "iTerm2", Event: "done", Count: "2",
	}
	tests := []struct {
		tmpl string
		want string
	}{
		{"{project}", "myproj"},
		{"{path}", "/repo"},
		{"{branch}", "main"},
		{"{repo}", "owner/myproj"},
		{"{model}", "Opus 5.5"},
		{"{context}", "84k / 1M"},
		{"{contextPercent}", "8%"},
		{"{duration}", "2m 14s"},
		{"{question}", "Pick one?"},
		{"{options}", "A, B"},
		{"{message}", "waiting"},
		{"{summary}", "Applied the fix."},
		{"{app}", "iTerm2"},
		{"{event}", "done"},
		{"{count}", "2"},
		{"{project}: {question}", "myproj: Pick one?"},
		{"no placeholders here", "no placeholders here"},
	}
	for _, tt := range tests {
		t.Run(tt.tmpl, func(t *testing.T) {
			if got := Render(tt.tmpl, v); got != tt.want {
				t.Errorf("Render(%q) = %q, want %q", tt.tmpl, got, tt.want)
			}
		})
	}
}

func TestRenderUnknownPlaceholderLeftAsWritten(t *testing.T) {
	got := Render("+{bogus} more", Vars{})
	if got != "+{bogus} more" {
		t.Errorf("Render() = %q, want unknown placeholder left as written", got)
	}
}

func TestRenderEmptyValuesTrimTrailingSeparator(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		v    Vars
		want string
	}{
		{
			name: "trailing colon-space trimmed",
			tmpl: "{project}: {question}",
			v:    Vars{Project: "myproj"},
			want: "myproj",
		},
		{
			name: "trailing comma trimmed",
			tmpl: "{project}, {branch}",
			v:    Vars{Project: "myproj"},
			want: "myproj",
		},
		{
			name: "empty value alone renders empty",
			tmpl: "{question}",
			v:    Vars{},
			want: "",
		},
		{
			name: "leading separator is not touched",
			tmpl: "{branch}: {project}",
			v:    Vars{Project: "myproj"},
			want: ": myproj",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Render(tt.tmpl, tt.v); got != tt.want {
				t.Errorf("Render(%q) = %q, want %q", tt.tmpl, got, tt.want)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1k"},
		{84000, "84k"},
		{84500, "84.5k"},
		{1_000_000, "1M"},
		{1_500_000, "1.5M"},
	}
	for _, tt := range tests {
		if got := FormatTokens(tt.in); got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatContext(t *testing.T) {
	if got := FormatContext(84000, 1_000_000); got != "84k / 1M" {
		t.Errorf("FormatContext() = %q, want %q", got, "84k / 1M")
	}
}

func TestFormatContextPercent(t *testing.T) {
	tests := []struct {
		used, window int
		want         string
	}{
		{84000, 1_000_000, "8%"},
		{100000, 200000, "50%"},
		{0, 0, "0%"},
	}
	for _, tt := range tests {
		if got := FormatContextPercent(tt.used, tt.window); got != tt.want {
			t.Errorf("FormatContextPercent(%d, %d) = %q, want %q", tt.used, tt.window, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0, "0s"},
		{5, "5s"},
		{59, "59s"},
		{60, "1m 0s"},
		{134, "2m 14s"},
		{3600, "1h 0m"},
		{3725, "1h 2m"},
		{-5, "0s"},
	}
	for _, tt := range tests {
		if got := FormatDuration(tt.seconds); got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}
