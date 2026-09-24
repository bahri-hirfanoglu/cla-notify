# cla-notify development

cla-notify is a Claude Code plugin that shows a desktop notification when Claude asks a
question, waits for approval or input, or finishes a turn. Clicking it returns to the
terminal the session runs in. It runs on macOS, Linux and Windows.

## Rules that apply to every change

- **English only.** Code, identifiers, comments, user-facing messages, config keys,
  docs, commit messages and PR text are all English.
- **No em dash (U+2014) or en dash (U+2013) anywhere**, in any file. Use a plain hyphen,
  a comma, a colon or a new sentence. `scripts/check-text.sh` enforces this and runs in
  `make check`.
- **Comments:** at most one sentence, and only for the "why" the code cannot say.
- **No AI attribution** in commits or PRs: no `Co-Authored-By` for an AI, no
  "Generated with" footers.
- **Commits:** one logical change each, imperative subject, a body that says what
  changed and why. Stage files by name, never `git add -A`.

## Layout

```
.claude-plugin/plugin.json      plugin manifest
.claude-plugin/marketplace.json this repository is its own marketplace
hooks/hooks.json                Claude Code hooks, all calling bin/cla-notify hook
commands/*.md                   /cla-notify:config, :test, :doctor
bin/cla-notify                  POSIX sh launcher: picks dist/<binary> for this OS/arch
dist/                           prebuilt binaries (built by scripts/build.sh, committed)
cmd/cla-notify/                 Go entry point
internal/card                   the Card JSON contract shared by every presenter
internal/config                 config file: load, defaults, merge, get/set, validate
internal/hook                   hook stdin parsing and event handling
internal/session                per-session state (turn start, model, terminal)
internal/transcript             transcript parsing: last answer, model, context usage
internal/gitinfo                branch, dirty flag, owner/repo remote
internal/render                 template placeholders -> final title/body/labels
internal/terminal               detect the terminal/tab the session runs in
internal/present                show a Card: darwin (Swift HUD), linux (D-Bus), windows (toast)
internal/focus                  bring the terminal back to the front on click
internal/paths                  config/state directories per OS
internal/cli                    subcommands
macos/hud/                      Swift renderer for macOS (cla-notify-hud)
scripts/                        build.sh, check-text.sh, e2e helpers
```

## The Card contract

`internal/card/card.go` is the single source of truth for what a presenter receives.
The Go core fills every field, including already-rendered text and display settings;
presenters never read the config file. `macos/hud/Card.swift` mirrors it field for
field. Changing the contract means changing both and bumping `card.Version`.

## Building and testing

Go is not assumed on the host. Use the container:

```
docker run --rm -v "$PWD":/src -w /src golang:1.25 go test ./...
```

`scripts/build.sh` cross-compiles every Go binary (darwin arm64+amd64 universal,
linux amd64/arm64, windows amd64) and builds the Swift HUD with `swiftc` on macOS.
Run `make check` (tests + vet for every GOOS + text check) before every commit.

Environment variables for testing: `CLA_NOTIFY_CONFIG` (config file path),
`CLA_NOTIFY_STATE_DIR` (state directory), `CLA_NOTIFY_DRYRUN=1` (print the Card JSON to
stdout instead of presenting it), `CLA_NOTIFY_HUD` (path to the macOS HUD binary).
