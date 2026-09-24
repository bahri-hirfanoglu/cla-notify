//go:build linux

// fakenotify is a real org.freedesktop.Notifications server for e2e-linux.sh: it logs every call
// and reacts like a notification daemon would, so RunLinuxWorker's wait loop resolves quickly.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	iface      = "org.freedesktop.Notifications"
	objectPath = "/org/freedesktop/Notifications"
	// clickMarker in a card's body tells fakenotify to simulate the user clicking the notification.
	clickMarker = "E2E_CLICK_MARKER"
)

type server struct {
	conn    *dbus.Conn
	logFile *os.File
	mu      sync.Mutex
	nextID  uint32
}

type notifyEntry struct {
	Type          string   `json:"type"`
	AppName       string   `json:"appName"`
	ReplacesID    uint32   `json:"replacesId"`
	Summary       string   `json:"summary"`
	Body          string   `json:"body"`
	Actions       []string `json:"actions"`
	Urgency       byte     `json:"urgency"`
	SoundName     string   `json:"soundName,omitempty"`
	SoundFile     string   `json:"soundFile,omitempty"`
	DesktopEntry  string   `json:"desktopEntry,omitempty"`
	ExpireTimeout int32    `json:"expireTimeout"`
	ID            uint32   `json:"id"`
}

type closeEntry struct {
	Type string `json:"type"`
	ID   uint32 `json:"id"`
}

func (s *server) log(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintln(s.logFile, string(data))
}

func (s *server) GetServerInformation() (string, string, string, string, *dbus.Error) {
	return "fake-notify", "cla-notify-e2e", "1.0", "1.2", nil
}

func (s *server) GetCapabilities() ([]string, *dbus.Error) {
	return []string{"body", "actions", "persistence", "sound"}, nil
}

func (s *server) Notify(appName string, replacesID uint32, appIcon string, summary string, body string, actions []string, hints map[string]dbus.Variant, expireTimeout int32) (uint32, *dbus.Error) {
	id := replacesID
	if id == 0 {
		s.mu.Lock()
		s.nextID++
		id = s.nextID
		s.mu.Unlock()
	}
	urgency, _ := hints["urgency"].Value().(byte)
	soundName, _ := hints["sound-name"].Value().(string)
	soundFile, _ := hints["sound-file"].Value().(string)
	desktopEntry, _ := hints["desktop-entry"].Value().(string)
	s.log(notifyEntry{
		Type: "notify", AppName: appName, ReplacesID: replacesID, Summary: summary, Body: body,
		Actions: actions, Urgency: urgency, SoundName: soundName, SoundFile: soundFile,
		DesktopEntry: desktopEntry, ExpireTimeout: expireTimeout, ID: id,
	})
	go s.react(id, body)
	return id, nil
}

func (s *server) CloseNotification(id uint32) *dbus.Error {
	s.log(closeEntry{Type: "close", ID: id})
	_ = s.conn.Emit(dbus.ObjectPath(objectPath), iface+".NotificationClosed", id, uint32(3))
	return nil
}

// react simulates a user action after a short delay: a click for a marked body, else a timeout close.
func (s *server) react(id uint32, body string) {
	if strings.Contains(body, clickMarker) {
		time.Sleep(300 * time.Millisecond)
		_ = s.conn.Emit(dbus.ObjectPath(objectPath), iface+".ActionInvoked", id, "focus")
		return
	}
	time.Sleep(2 * time.Second)
	_ = s.conn.Emit(dbus.ObjectPath(objectPath), iface+".NotificationClosed", id, uint32(1))
}

func main() {
	logPath := os.Getenv("FAKE_NOTIFY_LOG")
	if logPath == "" {
		fmt.Fprintln(os.Stderr, "FAKE_NOTIFY_LOG must be set")
		os.Exit(2)
	}
	f, err := os.Create(logPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create log:", err)
		os.Exit(1)
	}
	defer f.Close()

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect bus:", err)
		os.Exit(1)
	}
	defer conn.Close()

	s := &server{conn: conn, logFile: f}
	if err := conn.Export(s, dbus.ObjectPath(objectPath), iface); err != nil {
		fmt.Fprintln(os.Stderr, "export:", err)
		os.Exit(1)
	}
	reply, err := conn.RequestName(iface, dbus.NameFlagDoNotQueue)
	if err != nil {
		fmt.Fprintln(os.Stderr, "request name:", err)
		os.Exit(1)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "did not become primary owner of", iface)
		os.Exit(1)
	}

	fmt.Println("ready")
	select {}
}
