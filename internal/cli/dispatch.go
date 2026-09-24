package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/bahri-hirfanoglu/cla-notify/internal/card"
	"github.com/bahri-hirfanoglu/cla-notify/internal/paths"
	"github.com/bahri-hirfanoglu/cla-notify/internal/present"
)

func resolveConfigPath() (string, error) {
	return paths.ConfigPath()
}

// isDryRun reports whether a card should be printed instead of presented: CLA_NOTIFY_DRYRUN=1 or an explicit flag.
func isDryRun(flag bool) bool {
	return flag || os.Getenv("CLA_NOTIFY_DRYRUN") == "1"
}

// dispatchCard prints c as JSON in dry-run mode, otherwise hands it to the platform presenter.
func dispatchCard(c *card.Card, dryRun bool, out io.Writer) error {
	if isDryRun(dryRun) {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(c)
	}
	return present.ForOS().Show(c)
}
