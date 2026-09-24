//go:build linux

// Linux presenter: org.freedesktop.Notifications over the session bus.
package present

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/focus"
)

const (
	notifyDest = "org.freedesktop.Notifications"
	notifyPath = "/org/freedesktop/Notifications"
)

func init() {
	platform = linuxPresenter{}
}

type linuxPresenter struct{}

func (linuxPresenter) Name() string { return "linux-dbus" }

// Check reports whether the session bus is reachable and a notification service answers.
func (linuxPresenter) Check() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect session bus: %w", err)
	}
	defer conn.Close()
	obj := conn.Object(notifyDest, dbus.ObjectPath(notifyPath))
	var name, vendor, version, specVersion string
	if err := obj.Call(notifyDest+".GetServerInformation", 0).Store(&name, &vendor, &version, &specVersion); err != nil {
		return fmt.Errorf("notification service unavailable: %w", err)
	}
	return nil
}

// Show writes the card to the state dir and spawns a detached worker that presents it.
func (linuxPresenter) Show(c *card.Card) error {
	dir := filepath.Join(linuxStateDir(), "cards")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cards dir: %w", err)
	}
	id, err := randomID()
	if err != nil {
		return fmt.Errorf("generate card id: %w", err)
	}
	path := filepath.Join(dir, id+".json")
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode card: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write card: %w", err)
	}
	return runWorker(path)
}

// runWorker spawns the detached present-linux worker; overridden in tests to avoid a real spawn.
var runWorker = func(path string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate own executable: %w", err)
	}
	cmd := exec.Command(self, "present-linux", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn present-linux worker: %w", err)
	}
	return cmd.Process.Release()
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// linuxStateDir resolves the state directory the same way the spec's default does.
func linuxStateDir() string {
	if v := os.Getenv("CLA_NOTIFY_STATE_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "cla-notify")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "state", "cla-notify")
}

// RunLinuxWorker is the hidden present-linux subcommand: send the notification, wait, then clean up.
func RunLinuxWorker(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read card: %w", err)
	}
	var c card.Card
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("decode card: %w", err)
	}
	defer os.Remove(path)

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect session bus: %w", err)
	}
	defer conn.Close()

	obj := conn.Object(notifyDest, dbus.ObjectPath(notifyPath))

	replacesID, err := reserveNotification(c.SessionID)
	if err != nil {
		return fmt.Errorf("track notification id: %w", err)
	}

	actions := []string{"default", "", "focus", c.Labels.FocusButton}
	hints := notifyHints(&c)
	expire := expireTimeoutMillis(c.DurationSecs)

	call := obj.Call(notifyDest+".Notify", 0,
		"cla-notify", replacesID, "", c.Title, notifyBody(&c), actions, hints, expire)
	if call.Err != nil {
		return fmt.Errorf("notify: %w", call.Err)
	}
	var id uint32
	if err := call.Store(&id); err != nil {
		return fmt.Errorf("read notification id: %w", err)
	}

	closeFn := func(closeID uint32) {
		_ = obj.Call(notifyDest+".CloseNotification", 0, closeID).Err
	}
	if err := recordNotification(c.SessionID, id, c.Display.MaxCards, closeFn); err != nil {
		return fmt.Errorf("record notification: %w", err)
	}
	defer forgetNotification(c.SessionID, id)

	waitForClick(conn, id, c.Focus, c.DurationSecs)
	return nil
}

// waitForClick blocks until the notification is acted on, closed, or the timeout plus a margin passes.
func waitForClick(conn *dbus.Conn, id uint32, target card.FocusTarget, durationSecs float64) {
	ch := make(chan *dbus.Signal, 8)
	conn.Signal(ch)
	defer conn.RemoveSignal(ch)

	_ = conn.AddMatchSignal(dbus.WithMatchInterface(notifyDest), dbus.WithMatchMember("ActionInvoked"))
	_ = conn.AddMatchSignal(dbus.WithMatchInterface(notifyDest), dbus.WithMatchMember("NotificationClosed"))

	timeout := time.Duration(durationSecs*float64(time.Second)) + 5*time.Second
	if timeout <= 5*time.Second {
		timeout = 30 * time.Second
	}
	deadline := time.After(timeout)
	for {
		select {
		case sig, ok := <-ch:
			if !ok {
				return
			}
			if sig == nil || len(sig.Body) == 0 {
				continue
			}
			gotID, ok := sig.Body[0].(uint32)
			if !ok || gotID != id {
				continue
			}
			switch sig.Name {
			case notifyDest + ".ActionInvoked":
				_ = focus.Focus(target)
				return
			case notifyDest + ".NotificationClosed":
				return
			}
		case <-deadline:
			return
		}
	}
}

// notifyHints builds the urgency, sound and desktop-entry hints for a Notify call.
func notifyHints(c *card.Card) map[string]dbus.Variant {
	hints := map[string]dbus.Variant{
		"urgency":       dbus.MakeVariant(urgencyFor(c.Kind)),
		"desktop-entry": dbus.MakeVariant("cla-notify"),
	}
	switch sound := strings.TrimSpace(c.Sound); {
	case sound == "" || sound == "none":
		// silent, no sound hint
	case sound == "default":
		hints["sound-name"] = dbus.MakeVariant("message-new-instant")
	case strings.Contains(sound, "/"):
		hints["sound-file"] = dbus.MakeVariant(expandHome(sound))
	default:
		hints["sound-name"] = dbus.MakeVariant(sound)
	}
	return hints
}

func urgencyFor(k card.Kind) byte {
	if k == card.KindDone {
		return 1 // normal
	}
	return 2 // critical: ask and wait
}

func expandHome(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~"))
}

// expireTimeoutMillis converts seconds to the milliseconds Notify expects, -1 meaning the server default.
func expireTimeoutMillis(durationSecs float64) int32 {
	if durationSecs <= 0 {
		return -1
	}
	return int32(durationSecs * 1000)
}

// notifyBody appends a second line with project, branch and context, when the display settings ask for it.
func notifyBody(c *card.Card) string {
	body := c.Body
	var extra []string
	if c.Display.ShowGit {
		project := c.Project.Name
		if c.Project.Branch != "" {
			project = strings.TrimSpace(project + " (" + c.Project.Branch + ")")
		}
		if project != "" {
			extra = append(extra, project)
		}
	}
	if c.Display.ShowContext && c.Labels.Context != "" {
		extra = append(extra, c.Labels.Context)
	}
	if len(extra) == 0 {
		return body
	}
	return body + "\n" + strings.Join(extra, ", ")
}

// liveNotification is one of our own notifications tracked for replaces_id and maxCards.
type liveNotification struct {
	SessionID string `json:"sessionId"`
	NotifyID  uint32 `json:"notifyId"`
}

func liveListPath() string { return filepath.Join(linuxStateDir(), "linux-live.json") }
func lockPath() string     { return filepath.Join(linuxStateDir(), "linux-live.lock") }

// withLiveList reads, mutates and writes the tracked notification list under an exclusive file lock.
func withLiveList(fn func([]liveNotification) []liveNotification) error {
	if err := os.MkdirAll(linuxStateDir(), 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	var list []liveNotification
	if data, err := os.ReadFile(liveListPath()); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	list = fn(list)
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(liveListPath(), data, 0o600)
}

// reserveNotification returns the notify id of a still-live notification for this session, or 0.
func reserveNotification(sessionID string) (uint32, error) {
	var replaces uint32
	err := withLiveList(func(list []liveNotification) []liveNotification {
		for _, n := range list {
			if n.SessionID == sessionID {
				replaces = n.NotifyID
				break
			}
		}
		return list
	})
	return replaces, err
}

// recordNotification tracks id as the session's live notification and closes the oldest past maxCards.
func recordNotification(sessionID string, id uint32, maxCards int, closeFn func(uint32)) error {
	return withLiveList(func(list []liveNotification) []liveNotification {
		next := list[:0]
		for _, n := range list {
			if n.SessionID != sessionID {
				next = append(next, n)
			}
		}
		next = append(next, liveNotification{SessionID: sessionID, NotifyID: id})
		if maxCards > 0 {
			for len(next) > maxCards {
				oldest := next[0]
				closeFn(oldest.NotifyID)
				next = next[1:]
			}
		}
		return next
	})
}

// forgetNotification removes the tracked entry once its own notification has been closed or timed out.
func forgetNotification(sessionID string, id uint32) {
	_ = withLiveList(func(list []liveNotification) []liveNotification {
		next := list[:0]
		for _, n := range list {
			if n.SessionID == sessionID && n.NotifyID == id {
				continue
			}
			next = append(next, n)
		}
		return next
	})
}
