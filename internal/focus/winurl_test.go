package focus

import (
	"net/url"
	"path/filepath"
	"testing"
)

func TestStateDirPrefersOverride(t *testing.T) {
	getenv := func(k string) string {
		if k == "CLA_NOTIFY_STATE_DIR" {
			return "/custom"
		}
		return ""
	}
	if got := stateDir(getenv); got != "/custom" {
		t.Fatalf("stateDir() = %q, want /custom", got)
	}
}

func TestParseFocusURLValid(t *testing.T) {
	cardsDirPath := filepath.Join("state", "cards")
	cardPath := filepath.Join(cardsDirPath, "sess-1.json")
	raw := "cla-notify://focus?card=" + urlEscape(cardPath)

	got, err := parseFocusURL(raw, cardsDirPath)
	if err != nil {
		t.Fatalf("parseFocusURL() error = %v", err)
	}
	if got != filepath.Clean(cardPath) {
		t.Fatalf("parseFocusURL() = %q, want %q", got, cardPath)
	}
}

func TestParseFocusURLRejectsWrongScheme(t *testing.T) {
	if _, err := parseFocusURL("http://focus?card=x", "cards"); err == nil {
		t.Fatal("parseFocusURL() accepted a non cla-notify scheme")
	}
}

func TestParseFocusURLRejectsWrongHost(t *testing.T) {
	if _, err := parseFocusURL("cla-notify://open?card=x", "cards"); err == nil {
		t.Fatal("parseFocusURL() accepted a non focus host")
	}
}

func TestParseFocusURLRejectsMissingCard(t *testing.T) {
	if _, err := parseFocusURL("cla-notify://focus", "cards"); err == nil {
		t.Fatal("parseFocusURL() accepted a URL with no card param")
	}
}

func TestParseFocusURLRejectsOutsideCardsDir(t *testing.T) {
	cardsDirPath := filepath.Join("state", "cards")
	outside := filepath.Join("state", "..", "elsewhere", "x.json")
	raw := "cla-notify://focus?card=" + urlEscape(outside)
	if _, err := parseFocusURL(raw, cardsDirPath); err == nil {
		t.Fatal("parseFocusURL() accepted a path outside cardsDir")
	}
}

func TestParseFocusURLRejectsTraversal(t *testing.T) {
	cardsDirPath := filepath.Join("state", "cards")
	traversal := cardsDirPath + "/../../evil.json"
	raw := "cla-notify://focus?card=" + urlEscape(traversal)
	if _, err := parseFocusURL(raw, cardsDirPath); err == nil {
		t.Fatal("parseFocusURL() accepted a traversal path")
	}
}

func TestParseFocusURLRejectsCardsDirItself(t *testing.T) {
	cardsDirPath := filepath.Join("state", "cards")
	raw := "cla-notify://focus?card=" + urlEscape(cardsDirPath)
	if _, err := parseFocusURL(raw, cardsDirPath); err == nil {
		t.Fatal("parseFocusURL() accepted the cards dir itself")
	}
}

func urlEscape(s string) string {
	return url.QueryEscape(s)
}
