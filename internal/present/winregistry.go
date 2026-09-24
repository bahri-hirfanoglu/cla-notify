package present

import (
	"fmt"
	"strings"
)

// protocolKeyPath is the per-user registry key that registers the cla-notify:// URL protocol.
const protocolKeyPath = `Software\Classes\cla-notify`

// aumid is the Application User Model ID every notification and shortcut carries.
const aumid = "cla-notify"

// protocolCommand builds the shell command Windows runs when a cla-notify:// URL is activated.
// exePath is quoted as-is, so a double quote in it would break out of that quoting; refuse it instead.
func protocolCommand(exePath string) (string, error) {
	if strings.Contains(exePath, `"`) {
		return "", fmt.Errorf("exePath must not contain a double quote: %s", exePath)
	}
	return fmt.Sprintf(`"%s" focus-url "%%1"`, exePath), nil
}
