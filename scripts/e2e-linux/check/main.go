//go:build linux

// check asserts the e2e-linux.sh contract against the notify and tmux call logs, and exits non-zero on any failure.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type notifyEntry struct {
	Type          string   `json:"type"`
	AppName       string   `json:"appName"`
	ReplacesID    uint32   `json:"replacesId"`
	Summary       string   `json:"summary"`
	Body          string   `json:"body"`
	Actions       []string `json:"actions"`
	Urgency       byte     `json:"urgency"`
	SoundName     string   `json:"soundName"`
	SoundFile     string   `json:"soundFile"`
	DesktopEntry  string   `json:"desktopEntry"`
	ExpireTimeout int32    `json:"expireTimeout"`
	ID            uint32   `json:"id"`
}

type closeEntry struct {
	Type string `json:"type"`
	ID   uint32 `json:"id"`
}

var (
	passed int
	failed int
)

func check(name string, ok bool, detail string) {
	if ok {
		passed++
		fmt.Printf("PASS %s\n", name)
		return
	}
	failed++
	fmt.Printf("FAIL %s: %s\n", name, detail)
}

func main() {
	notifyLog := os.Getenv("FAKE_NOTIFY_LOG")
	tmuxLog := os.Getenv("TMUX_CALL_LOG")
	if notifyLog == "" || tmuxLog == "" {
		fmt.Fprintln(os.Stderr, "FAKE_NOTIFY_LOG and TMUX_CALL_LOG must be set")
		os.Exit(2)
	}

	notifies, closes := readNotifyLog(notifyLog)
	bySummary := map[string]notifyEntry{}
	for _, n := range notifies {
		bySummary[n.Summary] = n
	}

	check("received exactly 8 notify calls", len(notifies) == 8, fmt.Sprintf("got %d", len(notifies)))

	wantActions := []string{"default", "", "focus", "Back to Terminal"}
	for _, n := range notifies {
		check(n.Summary+": app name is cla-notify", n.AppName == "cla-notify", n.AppName)
		check(n.Summary+": desktop-entry hint is cla-notify", n.DesktopEntry == "cla-notify", n.DesktopEntry)
		check(n.Summary+": actions match", actionsEqual(n.Actions, wantActions), fmt.Sprintf("%v", n.Actions))
	}

	r1, ok1 := bySummary["cla-notify: approval needed"]
	r2, ok2 := bySummary["cla-notify: done in 12s"]
	check("replace-1 present", ok1, "missing")
	check("replace-2 present", ok2, "missing")
	if ok1 {
		check("replace-1: urgency critical", r1.Urgency == 2, fmt.Sprint(r1.Urgency))
		check("replace-1: expire timeout 4000ms", r1.ExpireTimeout == 4000, fmt.Sprint(r1.ExpireTimeout))
		check("replace-1: sound-name message-new-instant", r1.SoundName == "message-new-instant", r1.SoundName)
		check("replace-1: replaces_id is 0 (first notification)", r1.ReplacesID == 0, fmt.Sprint(r1.ReplacesID))
	}
	if ok1 && ok2 {
		check("replace-2: urgency normal", r2.Urgency == 1, fmt.Sprint(r2.Urgency))
		check("replace-2: expire timeout 1000ms", r2.ExpireTimeout == 1000, fmt.Sprint(r2.ExpireTimeout))
		check("replace-2: silent, no sound hints", r2.SoundName == "" && r2.SoundFile == "", r2.SoundName+"/"+r2.SoundFile)
		check("replace-2: replaces_id equals replace-1's id", r2.ReplacesID == r1.ID, fmt.Sprintf("got %d want %d", r2.ReplacesID, r1.ID))
		check("replace-2: id reused from replace-1", r2.ID == r1.ID, fmt.Sprintf("got %d want %d", r2.ID, r1.ID))
	}

	m1, okm1 := bySummary["Max Card 1"]
	m2, okm2 := bySummary["Max Card 2"]
	m3, okm3 := bySummary["Max Card 3"]
	m4, okm4 := bySummary["Max Card 4"]
	for name, ok := range map[string]bool{"max-1": okm1, "max-2": okm2, "max-3": okm3, "max-4": okm4} {
		check(name+" present", ok, "missing")
	}
	if okm1 && okm2 && okm3 && okm4 {
		check("max cards: expire timeout 3000ms each", m1.ExpireTimeout == 3000 && m2.ExpireTimeout == 3000 && m3.ExpireTimeout == 3000 && m4.ExpireTimeout == 3000, "mismatch")
		check("max-1 (wait): urgency critical", m1.Urgency == 2, fmt.Sprint(m1.Urgency))
		check("max-2 (ask): urgency critical", m2.Urgency == 2, fmt.Sprint(m2.Urgency))
		check("max-3 (wait): urgency critical", m3.Urgency == 2, fmt.Sprint(m3.Urgency))
		check("max-4 (done): urgency normal", m4.Urgency == 1, fmt.Sprint(m4.Urgency))
		check("max-2: sound-name bell passthrough", m2.SoundName == "bell", m2.SoundName)
		check("max-3: sound-file path", m3.SoundFile == "/opt/sounds/ding.oga", m3.SoundFile)
		check("max-4: silent, no sound hints", m4.SoundName == "" && m4.SoundFile == "", m4.SoundName+"/"+m4.SoundFile)

		check("maxCards: exactly one close call", len(closes) == 1, fmt.Sprintf("got %d", len(closes)))
		if len(closes) == 1 {
			check("maxCards: closed the oldest (max-1)", closes[0].ID == m1.ID, fmt.Sprintf("closed %d want %d", closes[0].ID, m1.ID))
		}
	}

	click, okc := bySummary["cla-notify: Claude has a question"]
	check("click card present", okc, "missing")
	if okc {
		wantBody := "Pick one E2E_CLICK_MARKER\ncla-notify (main), 12k / 200k"
		check("click: body includes project/branch/context line", click.Body == wantBody, click.Body)
		check("click: expire timeout 5000ms", click.ExpireTimeout == 5000, fmt.Sprint(click.ExpireTimeout))
	}

	hookNotify, okhook := bySummary["project: approval needed"]
	check("hook scenario: notify present", okhook, "missing")
	if okhook {
		check("hook scenario: body has rendered message", strings.Contains(hookNotify.Body, "Claude wants to delete a file."), hookNotify.Body)
	}

	tmuxCalls := readLines(tmuxLog)
	wantTmux := "tmux -S /tmp/e2e-tmux.sock select-pane -t %3"
	check("ActionInvoked(focus) triggered tmux select-pane", containsLine(tmuxCalls, wantTmux), strings.Join(tmuxCalls, " | "))

	fmt.Printf("\n== e2e-linux: %d passed, %d failed ==\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func actionsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func readNotifyLog(path string) ([]notifyEntry, []closeEntry) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open notify log:", err)
		os.Exit(1)
	}
	defer f.Close()

	var notifies []notifyEntry
	var closes []closeEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var kind struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &kind); err != nil {
			continue
		}
		switch kind.Type {
		case "notify":
			var n notifyEntry
			if err := json.Unmarshal([]byte(line), &n); err == nil {
				notifies = append(notifies, n)
			}
		case "close":
			var c closeEntry
			if err := json.Unmarshal([]byte(line), &c); err == nil {
				closes = append(closes, c)
			}
		}
	}
	return notifies, closes
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func containsLine(lines []string, want string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == want {
			return true
		}
	}
	return false
}
