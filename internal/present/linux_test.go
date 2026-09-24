//go:build linux

package present

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

func TestLinuxPresenterName(t *testing.T) {
	if got := (linuxPresenter{}).Name(); got != "linux-dbus" {
		t.Fatalf("Name() = %q, want linux-dbus", got)
	}
}

func TestLinuxPresenterRegistered(t *testing.T) {
	if _, ok := ForOS().(linuxPresenter); !ok {
		t.Fatalf("ForOS() = %T, want linuxPresenter", ForOS())
	}
}

func TestCheckWithoutBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/does-not-exist")
	if err := (linuxPresenter{}).Check(); err == nil {
		t.Fatal("Check() = nil, want an error when the session bus is unreachable")
	}
}

func TestShowWritesCardAndSpawnsWorker(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CLA_NOTIFY_STATE_DIR", stateDir)

	var spawned string
	old := runWorker
	runWorker = func(path string) error {
		spawned = path
		return nil
	}
	t.Cleanup(func() { runWorker = old })

	c := &card.Card{SessionID: "sess-1", Title: "Title", Body: "Body"}
	if err := (linuxPresenter{}).Show(c); err != nil {
		t.Fatalf("Show() error = %v", err)
	}

	if spawned == "" {
		t.Fatal("runWorker was not called")
	}
	if filepath.Dir(spawned) != filepath.Join(stateDir, "cards") {
		t.Fatalf("card written to %q, want under %q/cards", spawned, stateDir)
	}
	data, err := os.ReadFile(spawned)
	if err != nil {
		t.Fatalf("read written card: %v", err)
	}
	var got card.Card
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written card: %v", err)
	}
	if got.SessionID != "sess-1" || got.Title != "Title" {
		t.Fatalf("written card = %+v, want sessionId sess-1 and title Title", got)
	}
}

func TestLinuxStateDirDefaults(t *testing.T) {
	t.Setenv("CLA_NOTIFY_STATE_DIR", "")
	t.Setenv("XDG_STATE_HOME", "/xdg-state")
	if got := linuxStateDir(); got != "/xdg-state/cla-notify" {
		t.Fatalf("linuxStateDir() = %q, want /xdg-state/cla-notify", got)
	}

	t.Setenv("XDG_STATE_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	want := filepath.Join(home, ".local", "state", "cla-notify")
	if got := linuxStateDir(); got != want {
		t.Fatalf("linuxStateDir() = %q, want %q", got, want)
	}
}

func TestNotifyHints(t *testing.T) {
	cases := []struct {
		name       string
		sound      string
		wantHint   string
		wantValue  any
		wantAbsent []string
	}{
		{"empty is silent", "", "", nil, []string{"sound-name", "sound-file"}},
		{"none is silent", "none", "", nil, []string{"sound-name", "sound-file"}},
		{"default maps to message-new-instant", "default", "sound-name", "message-new-instant", []string{"sound-file"}},
		{"path becomes sound-file", "/usr/share/sounds/x.oga", "sound-file", "/usr/share/sounds/x.oga", []string{"sound-name"}},
		{"plain name becomes sound-name", "bell", "sound-name", "bell", []string{"sound-file"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &card.Card{Sound: tc.sound}
			hints := notifyHints(c)
			if got := hints["desktop-entry"].Value(); got != "cla-notify" {
				t.Fatalf("desktop-entry = %v, want cla-notify", got)
			}
			for _, absent := range tc.wantAbsent {
				if _, ok := hints[absent]; ok {
					t.Fatalf("hint %q present, want absent", absent)
				}
			}
			if tc.wantHint == "" {
				return
			}
			v, ok := hints[tc.wantHint]
			if !ok {
				t.Fatalf("hint %q missing", tc.wantHint)
			}
			if v.Value() != tc.wantValue {
				t.Fatalf("hint %q = %v, want %v", tc.wantHint, v.Value(), tc.wantValue)
			}
		})
	}
}

func TestUrgencyFor(t *testing.T) {
	if urgencyFor(card.KindAsk) != 2 {
		t.Fatal("ask should be critical")
	}
	if urgencyFor(card.KindWait) != 2 {
		t.Fatal("wait should be critical")
	}
	if urgencyFor(card.KindDone) != 1 {
		t.Fatal("done should be normal")
	}
}

func TestExpireTimeoutMillis(t *testing.T) {
	if got := expireTimeoutMillis(0); got != -1 {
		t.Fatalf("expireTimeoutMillis(0) = %d, want -1", got)
	}
	if got := expireTimeoutMillis(8); got != 8000 {
		t.Fatalf("expireTimeoutMillis(8) = %d, want 8000", got)
	}
}

func TestNotifyBody(t *testing.T) {
	c := &card.Card{Body: "Question text"}
	if got := notifyBody(c); got != "Question text" {
		t.Fatalf("notifyBody() = %q, want unchanged body when display flags are off", got)
	}

	c = &card.Card{
		Body:    "Question text",
		Project: card.Project{Name: "cla-notify", Branch: "main"},
		Labels:  card.Labels{Context: "12k / 200k"},
		Display: card.Display{ShowGit: true, ShowContext: true},
	}
	want := "Question text\ncla-notify (main), 12k / 200k"
	if got := notifyBody(c); got != want {
		t.Fatalf("notifyBody() = %q, want %q", got, want)
	}
}

func TestLiveListReserveRecordForget(t *testing.T) {
	t.Setenv("CLA_NOTIFY_STATE_DIR", t.TempDir())

	if id, err := reserveNotification("s1"); err != nil || id != 0 {
		t.Fatalf("reserveNotification(new) = (%d, %v), want (0, nil)", id, err)
	}

	noop := func(uint32) {}
	if err := recordNotification("s1", 10, 3, noop); err != nil {
		t.Fatalf("recordNotification: %v", err)
	}
	if id, err := reserveNotification("s1"); err != nil || id != 10 {
		t.Fatalf("reserveNotification(existing) = (%d, %v), want (10, nil)", id, err)
	}

	// Replacing the same session should not grow the live count.
	if err := recordNotification("s1", 11, 3, noop); err != nil {
		t.Fatalf("recordNotification replace: %v", err)
	}

	if err := recordNotification("s2", 12, 3, noop); err != nil {
		t.Fatal(err)
	}
	if err := recordNotification("s3", 13, 3, noop); err != nil {
		t.Fatal(err)
	}

	var closed []uint32
	closeFn := func(id uint32) { closed = append(closed, id) }
	if err := recordNotification("s4", 14, 3, closeFn); err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0] != 11 {
		t.Fatalf("closed = %v, want the oldest live id [11]", closed)
	}

	forgetNotification("s3", 13)
	if id, err := reserveNotification("s3"); err != nil || id != 0 {
		t.Fatalf("reserveNotification(forgotten) = (%d, %v), want (0, nil)", id, err)
	}
}
