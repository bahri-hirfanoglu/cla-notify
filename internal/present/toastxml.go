// Package present: pure WinRT toast XML building, tested on every OS.
package present

import (
	"bytes"
	"encoding/xml"
	"net/url"
	"strings"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
)

// toastGroup is the WinRT notification group every cla-notify toast belongs to.
const toastGroup = "cla-notify"

// focusURL builds the cla-notify:// URL a toast click or its action button activates.
func focusURL(cardPath string) string {
	return "cla-notify://focus?card=" + url.QueryEscape(cardPath)
}

// toastTag derives a per-session toast tag so a new card from the same session replaces the last one.
func toastTag(sessionID string) string {
	var b strings.Builder
	for _, r := range sessionID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	tag := b.String()
	if tag == "" {
		tag = "session"
	}
	if len(tag) > 64 {
		tag = tag[:64]
	}
	return tag
}

// attributionText renders the project and branch line shown under the toast body.
func attributionText(p card.Project) string {
	if p.Branch == "" {
		return p.Name
	}
	return p.Name + " on " + p.Branch
}

// toastAudio is the resolved <audio> element for a toast.
type toastAudio struct {
	src    string
	silent bool
}

// filePathToFileURI turns a Windows file path into a file:// URI, independent of the host OS.
func filePathToFileURI(p string) string {
	slash := strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return (&url.URL{Scheme: "file", Path: slash}).String()
}

// resolveToastAudio maps a card's resolved sound value to the toast XML audio element.
func resolveToastAudio(sound string) toastAudio {
	switch sound {
	case "default":
		return toastAudio{src: "ms-winsoundevent:Notification.Default"}
	case "none", "":
		return toastAudio{silent: true}
	default:
		if strings.HasPrefix(sound, "ms-appx:") || strings.HasPrefix(sound, "file:") {
			return toastAudio{src: sound}
		}
		return toastAudio{src: filePathToFileURI(sound)}
	}
}

// escapeXML makes s safe to place inside a toast XML text node or attribute value.
func escapeXML(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// buildToastXML renders the WinRT toast XML for c; targetURL is used for both the toast's
// own launch and its action button.
func buildToastXML(c *card.Card, targetURL string) string {
	var b strings.Builder
	b.WriteString(`<toast launch="`)
	b.WriteString(escapeXML(targetURL))
	b.WriteString(`"><visual><binding template="ToastGeneric">`)
	b.WriteString("<text>" + escapeXML(c.Title) + "</text>")
	b.WriteString("<text>" + escapeXML(c.Body) + "</text>")
	if attribution := attributionText(c.Project); attribution != "" {
		b.WriteString(`<text placement="attribution">` + escapeXML(attribution) + "</text>")
	}
	if c.Display.ShowContext && c.Labels.Context != "" {
		b.WriteString("<text>" + escapeXML(c.Labels.Context) + "</text>")
	}
	b.WriteString("</binding></visual>")
	b.WriteString(`<actions><action content="`)
	b.WriteString(escapeXML(c.Labels.FocusButton))
	b.WriteString(`" arguments="`)
	b.WriteString(escapeXML(targetURL))
	b.WriteString(`" activationType="protocol"/></actions>`)
	audio := resolveToastAudio(c.Sound)
	if audio.silent {
		b.WriteString(`<audio silent="true"/>`)
	} else {
		b.WriteString(`<audio src="` + escapeXML(audio.src) + `"/>`)
	}
	b.WriteString("</toast>")
	return b.String()
}
