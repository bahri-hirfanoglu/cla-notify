package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/session"
)

func testDeps(t *testing.T, now time.Time) Deps {
	t.Helper()
	return Deps{
		ConfigPath: filepath.Join(t.TempDir(), "missing-config.json"),
		StateDir:   t.TempDir(),
		Now:        func() time.Time { return now },
		Env:        map[string]string{},
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestProcessSessionStartRecordsModelNoCard(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "sessionstart.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(SessionStart) = %+v, want no card", c)
	}
	st := session.Load(deps.StateDir, "sess-1")
	if st.Model != "claude-opus-5-5" {
		t.Errorf("session Model = %q, want claude-opus-5-5", st.Model)
	}
}

func TestProcessUserPromptSubmitRecordsTurnStart(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	deps := testDeps(t, now)
	c, err := Process(readFixture(t, "userpromptsubmit.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(UserPromptSubmit) = %+v, want no card", c)
	}
	st := session.Load(deps.StateDir, "sess-1")
	if st.TurnStartedAt == nil || !st.TurnStartedAt.Equal(now) {
		t.Errorf("TurnStartedAt = %v, want %v", st.TurnStartedAt, now)
	}
}

func TestProcessAskUserQuestion(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "ask.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c == nil {
		t.Fatal("Process(ask) = nil, want a card")
	}
	if c.Kind != card.KindAsk {
		t.Errorf("Kind = %q, want ask", c.Kind)
	}
	if c.Event != "ask" {
		t.Errorf("Event = %q, want ask", c.Event)
	}
	if c.Body != "Which approach do you prefer?" {
		t.Errorf("Body = %q", c.Body)
	}
	if len(c.Options) != 2 || c.Options[0] != "Keep current structure" {
		t.Errorf("Options = %v", c.Options)
	}
	if c.QuestionCount != 1 {
		t.Errorf("QuestionCount = %d, want 1", c.QuestionCount)
	}
	if c.Project.Name != "project" {
		t.Errorf("Project.Name = %q, want project", c.Project.Name)
	}

	st := session.Load(deps.StateDir, "sess-1")
	if st.AskedAt == nil {
		t.Error("AskedAt should be set after an ask card")
	}
}

func TestProcessAskUserQuestionMoreQuestionsLabel(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "ask.json"), deps)
	if err != nil {
		t.Fatalf("Process(ask) error = %v", err)
	}
	if c == nil {
		t.Fatal("Process(ask) = nil, want a card")
	}
	if c.QuestionCount != 1 {
		t.Errorf("QuestionCount = %d, want 1", c.QuestionCount)
	}
	if c.Labels.MoreQuestions != "" {
		t.Errorf("Labels.MoreQuestions = %q, want empty for a single question", c.Labels.MoreQuestions)
	}

	deps3 := testDeps(t, time.Now())
	c3, err := Process(readFixture(t, "ask3.json"), deps3)
	if err != nil {
		t.Fatalf("Process(ask3) error = %v", err)
	}
	if c3 == nil {
		t.Fatal("Process(ask3) = nil, want a card")
	}
	if c3.QuestionCount != 3 {
		t.Errorf("QuestionCount = %d, want 3", c3.QuestionCount)
	}
	if want := "+2 more questions"; c3.Labels.MoreQuestions != want {
		t.Errorf("Labels.MoreQuestions = %q, want %q", c3.Labels.MoreQuestions, want)
	}
}

func TestProcessAskUserQuestionDisabled(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"events":{"ask":{"enabled":false}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	deps := Deps{ConfigPath: cfgPath, StateDir: t.TempDir(), Now: time.Now, Env: map[string]string{}}
	c, err := Process(readFixture(t, "ask.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(ask) with events.ask.enabled=false = %+v, want no card", c)
	}
}

func TestProcessNotificationPermission(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "permission.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c == nil {
		t.Fatal("Process(permission) = nil, want a card")
	}
	if c.Kind != card.KindWait || c.Event != "permission" {
		t.Errorf("Kind/Event = %s/%s, want wait/permission", c.Kind, c.Event)
	}
	if c.Body != "Claude wants to delete a file." {
		t.Errorf("Body = %q", c.Body)
	}
}

func TestProcessNotificationIdle(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "idle.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c == nil {
		t.Fatal("Process(idle) = nil, want a card")
	}
	if c.Kind != card.KindWait || c.Event != "idle" {
		t.Errorf("Kind/Event = %s/%s, want wait/idle", c.Kind, c.Event)
	}
}

func TestProcessNotificationSuppressedRightAfterAsk(t *testing.T) {
	deps := testDeps(t, time.Now())
	if _, err := Process(readFixture(t, "ask.json"), deps); err != nil {
		t.Fatalf("Process(ask) error = %v", err)
	}
	c, err := Process(readFixture(t, "permission.json"), deps)
	if err != nil {
		t.Fatalf("Process(permission) error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(permission) right after ask = %+v, want suppressed", c)
	}
}

func TestProcessStopUsesTurnStartedAt(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	promptDeps := testDeps(t, t0)
	if _, err := Process(readFixture(t, "userpromptsubmit.json"), promptDeps); err != nil {
		t.Fatalf("Process(UserPromptSubmit) error = %v", err)
	}

	t1 := t0.Add(134 * time.Second)
	stopDeps := promptDeps
	stopDeps.Now = func() time.Time { return t1 }
	c, err := Process(readFixture(t, "stop.json"), stopDeps)
	if err != nil {
		t.Fatalf("Process(Stop) error = %v", err)
	}
	if c == nil {
		t.Fatal("Process(Stop) = nil, want a card")
	}
	if c.Kind != card.KindDone || c.Event != "done" {
		t.Errorf("Kind/Event = %s/%s, want done/done", c.Kind, c.Event)
	}
	if c.ElapsedSeconds == nil || *c.ElapsedSeconds != 134 {
		t.Errorf("ElapsedSeconds = %v, want 134", c.ElapsedSeconds)
	}
	if c.Model != "Opus 5.5" {
		t.Errorf("Model = %q, want Opus 5.5", c.Model)
	}
	if c.Context == nil || c.Context.UsedTokens != 84000 {
		t.Errorf("Context = %+v, want UsedTokens 84000", c.Context)
	}

	st := session.Load(stopDeps.StateDir, "sess-1")
	if st.TurnStartedAt != nil || st.AskedAt != nil {
		t.Errorf("Stop should clear turn state, got %+v", st)
	}
}

func TestProcessStopFallsBackToTranscriptTimestamp(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	deps := testDeps(t, t1)
	c, err := Process(readFixture(t, "stop.json"), deps)
	if err != nil {
		t.Fatalf("Process(Stop) error = %v", err)
	}
	if c == nil {
		t.Fatal("Process(Stop) = nil, want a card")
	}
	// The transcript's last user timestamp is 2024-01-01T00:00:00Z (12h before t1: 43200s).
	if c.ElapsedSeconds == nil || *c.ElapsedSeconds != 43200 {
		t.Errorf("ElapsedSeconds = %v, want 43200", c.ElapsedSeconds)
	}
}

func TestProcessStopHookActiveSkipped(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "stop_active.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(Stop, stop_hook_active=true) = %+v, want no card", c)
	}
}

func TestProcessStopMinTurnSecondsSuppresses(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"events":{"done":{"minTurnSeconds":60}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	stateDir := t.TempDir()
	promptDeps := Deps{ConfigPath: cfgPath, StateDir: stateDir, Now: func() time.Time { return t0 }, Env: map[string]string{}}
	if _, err := Process(readFixture(t, "userpromptsubmit.json"), promptDeps); err != nil {
		t.Fatal(err)
	}

	stopDeps := promptDeps
	stopDeps.Now = func() time.Time { return t0.Add(10 * time.Second) }
	c, err := Process(readFixture(t, "stop.json"), stopDeps)
	if err != nil {
		t.Fatalf("Process(Stop) error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(Stop) elapsed=10s with minTurnSeconds=60 = %+v, want suppressed", c)
	}
}

func TestProcessDisabledConfigSuppressesEverything(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"enabled":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	deps := Deps{ConfigPath: cfgPath, StateDir: t.TempDir(), Now: time.Now, Env: map[string]string{}}
	c, err := Process(readFixture(t, "ask.json"), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Errorf("Process() with enabled=false = %+v, want no card", c)
	}
}

func TestProcessUnknownEventNoCard(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process([]byte(`{"hook_event_name":"SomethingElse","session_id":"sess-1"}`), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Errorf("Process(unknown event) = %+v, want no card", c)
	}
}

func TestProcessMissingEventNameIsIgnored(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process([]byte(`{}`), deps)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if c != nil {
		t.Error("Process({}) should return no card and no error")
	}
}

func TestProcessMalformedJSONReturnsError(t *testing.T) {
	deps := testDeps(t, time.Now())
	if _, err := Process([]byte(`{not json`), deps); err == nil {
		t.Error("Process() expected an error for malformed JSON")
	}
}

func TestCardWithoutOptionsEncodesEmptyArray(t *testing.T) {
	deps := testDeps(t, time.Now())
	c, err := Process(readFixture(t, "idle.json"), deps)
	if err != nil || c == nil {
		t.Fatalf("Process(idle) = %v, %v", c, err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"options":[]`) {
		t.Errorf("card JSON lacks \"options\":[], the macOS HUD rejects null: %s", data)
	}
}
