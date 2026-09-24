# Contributing to cla-notify

Thanks for considering a contribution. This project keeps a small surface and a
strict style, described in `CLAUDE.md`. Read that file first, it is the single
source of truth for the rules; this file covers the mechanics of getting a change
merged.

## Dev setup

Go is not assumed on the host. If you have a local `go`, the Makefile uses it;
otherwise it falls back to a container:

```
docker run --rm -v "$PWD":/src -w /src golang:1.25 go test ./...
```

The Swift HUD (`macos/hud/`, the `cla-notify-hud` binary) only builds on macOS, with
`swiftc` from a local Xcode Command Line Tools install. `scripts/build-hud.sh` builds
it; on Linux and Windows that step is skipped.

## Running the checks

```
make check      # go test, go vet for every target OS, and the no-dash text check
make e2e-linux   # Linux end-to-end script
```

Run `make check` before every commit. CI (`.github/workflows/ci.yml`) runs the same
matrix plus the macOS HUD build and a Windows smoke test, and must pass before a PR
merges.

## Testing cards safely

Do not rely on a live notification daemon or the HUD to check your work. Set
`CLA_NOTIFY_DRYRUN=1` and cla-notify prints the Card JSON to stdout instead of
presenting it:

```
CLA_NOTIFY_DRYRUN=1 ./cla-notify test ask
```

To see the actual rendered card on macOS, run the HUD binary directly against a
card file:

```
./dist/cla-notify-hud render path/to/card.json
```

Use a temporary `CLA_NOTIFY_CONFIG` and `CLA_NOTIFY_STATE_DIR` when experimenting so
you do not touch your own configuration.

## The Card contract

`internal/card/card.go` is the single source of truth for the JSON every presenter
receives. `macos/hud/Card.swift` mirrors it field for field. If your change adds,
removes or renames a field:

1. Update both `card.go` and `Card.swift` in the same commit.
2. Bump `card.Version`.
3. Update any presenter that reads the changed field.

A PR that changes one side without the other will not be merged.

## Commit and PR rules

Follow `CLAUDE.md`:

- One logical change per commit, imperative subject line, a body explaining what
  changed and why.
- Stage files by name (`git add <file>`), never `git add -A`.
- No AI attribution anywhere: no `Co-Authored-By` for an AI, no "Generated with"
  footers.
- English only, no em dash (U+2014) or en dash (U+2013) in any file. Use a hyphen,
  comma, colon or a new sentence instead. `scripts/check-text.sh` enforces this.
- Comments: at most one sentence, and only for the "why" the code cannot say.

## Workflow

1. Fork the repository.
2. Create a branch off `main` for your change.
3. Open a pull request against `main`. Describe what changed and why, and note
   which platforms you tested on.
4. CI must pass on all three operating systems before review.
5. A maintainer reviews and merges.

## dist/ and releases

`dist/` holds prebuilt binaries and is committed, but contributors must not commit
changes to it. Regenerating `dist/` with `scripts/build.sh` is done by the
maintainer as part of cutting a release. A PR that only touches `dist/` output
will be asked to drop that change.
