## Summary

<!-- What does this PR change, and why? -->

## Platforms tested

<!-- macOS / Linux / Windows, and how (make check, e2e-linux, manual test, dry run) -->

## Checklist

- [ ] `make check` passes locally
- [ ] No em dash (U+2014) or en dash (U+2013) added anywhere
- [ ] If the Card contract changed, `internal/card/card.go` and `macos/hud/Card.swift`
      were updated together and `card.Version` was bumped
- [ ] Docs updated (`README.md`, `CLAUDE.md`, `config.example.json`) if behavior or
      config changed
