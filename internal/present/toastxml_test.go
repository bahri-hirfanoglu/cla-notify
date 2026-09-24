package present

import (
	"strings"
	"testing"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func TestFocusURLEscapesPath(t *testing.T) {
	got := focusURL(`C:\Users\alice\AppData\Local\cla-notify\cards\abc def.json`)
	want := "cla-notify://focus?card=C%3A%5CUsers%5Calice%5CAppData%5CLocal%5Ccla-notify%5Ccards%5Cabc+def.json"
	if got != want {
		t.Fatalf("focusURL() = %q, want %q", got, want)
	}
}

func TestToastTagStripsUnsafeChars(t *testing.T) {
	got := toastTag("sess-123_ABC!@#")
	want := "sess123ABC"
	if got != want {
		t.Fatalf("toastTag() = %q, want %q", got, want)
	}
}

func TestToastTagFallsBackWhenEmpty(t *testing.T) {
	if got := toastTag("---"); got != "session" {
		t.Fatalf("toastTag() = %q, want %q", got, "session")
	}
}

func TestToastTagTruncates(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := toastTag(long)
	if len(got) != 64 {
		t.Fatalf("toastTag() length = %d, want 64", len(got))
	}
}

func TestAttributionText(t *testing.T) {
	cases := []struct {
		p    card.Project
		want string
	}{
		{card.Project{Name: "myapp"}, "myapp"},
		{card.Project{Name: "myapp", Branch: "main"}, "myapp on main"},
	}
	for _, tc := range cases {
		if got := attributionText(tc.p); got != tc.want {
			t.Errorf("attributionText(%+v) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestFilePathToFileURI(t *testing.T) {
	got := filePathToFileURI(`C:\Users\alice\sound.wav`)
	want := "file:///C:/Users/alice/sound.wav"
	if got != want {
		t.Fatalf("filePathToFileURI() = %q, want %q", got, want)
	}
}

func TestResolveToastAudio(t *testing.T) {
	cases := []struct {
		sound      string
		wantSilent bool
		wantSrc    string
	}{
		{"default", false, "ms-winsoundevent:Notification.Default"},
		{"none", true, ""},
		{"", true, ""},
		{"ms-appx:///sounds/x.wav", false, "ms-appx:///sounds/x.wav"},
		{"file:///C:/x.wav", false, "file:///C:/x.wav"},
		{`C:\x.wav`, false, "file:///C:/x.wav"},
	}
	for _, tc := range cases {
		got := resolveToastAudio(tc.sound)
		if got.silent != tc.wantSilent || got.src != tc.wantSrc {
			t.Errorf("resolveToastAudio(%q) = %+v, want silent=%v src=%q", tc.sound, got, tc.wantSilent, tc.wantSrc)
		}
	}
}

func TestEscapeXML(t *testing.T) {
	got := escapeXML(`<a> & "b" 'c'`)
	if strings.ContainsAny(got, "<>") {
		t.Fatalf("escapeXML left raw angle brackets: %q", got)
	}
}

func TestBuildToastXMLIncludesCoreFields(t *testing.T) {
	c := &card.Card{
		Title:   "myapp: Claude has a question",
		Body:    "Should I proceed?",
		Project: card.Project{Name: "myapp", Branch: "main"},
		Sound:   "default",
		Labels:  card.Labels{FocusButton: "Back to Windows Terminal", Context: "84k / 1M"},
		Display: card.Display{ShowContext: true},
	}
	xmlStr := buildToastXML(c, "cla-notify://focus?card=x")
	for _, want := range []string{
		"<toast launch=\"cla-notify://focus?card=x\">",
		"myapp: Claude has a question",
		"Should I proceed?",
		"myapp on main",
		"84k / 1M",
		"activationType=\"protocol\"",
		"ms-winsoundevent:Notification.Default",
		"Back to Windows Terminal",
	} {
		if !strings.Contains(xmlStr, want) {
			t.Errorf("buildToastXML() missing %q in:\n%s", want, xmlStr)
		}
	}
}

func TestBuildToastXMLSilentAudio(t *testing.T) {
	c := &card.Card{Sound: "none"}
	xmlStr := buildToastXML(c, "cla-notify://focus?card=x")
	if !strings.Contains(xmlStr, `<audio silent="true"/>`) {
		t.Fatalf("buildToastXML() missing silent audio element: %s", xmlStr)
	}
}

func TestBuildToastXMLOmitsContextWhenDisabled(t *testing.T) {
	c := &card.Card{Labels: card.Labels{Context: "84k / 1M"}, Display: card.Display{ShowContext: false}}
	xmlStr := buildToastXML(c, "cla-notify://focus?card=x")
	if strings.Contains(xmlStr, "84k / 1M") {
		t.Fatalf("buildToastXML() included context text despite ShowContext=false: %s", xmlStr)
	}
}
