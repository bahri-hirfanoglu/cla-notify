# Security Policy

## Supported Versions

cla-notify is pre-1.0. Only the latest released `0.x` version is supported with
security fixes; there is no long-term support branch.

| Version | Supported |
|---|---|
| latest 0.x | yes |
| older 0.x | no |

## Reporting a Vulnerability

Report a suspected vulnerability privately, not in a public issue:

1. Open a
   [GitHub security advisory](https://github.com/bahri-hirfanoglu/cla-notify/security/advisories/new)
   on this repository, or
2. Email the maintainer at the address listed on
   [their GitHub profile](https://github.com/bahri-hirfanoglu).

Include the affected version (`cla-notify version`), the platform, and steps to
reproduce. You should get an initial response within a few days.

## Scope

The following areas are considered security-sensitive and are in scope for
reports:

- **Hook input handling**: how `cla-notify hook` parses stdin from Claude Code
  and derives state, config and session data from it.
- **The `cla-notify://` URL handler**: how `cla-notify focus`/`focus-url`
  resolves a card path and terminal target from an incoming URL, including path
  traversal outside the state directory's cards folder.
- **AppleScript focus**: the macOS focus mechanism that brings a terminal window
  to the front, and anything it is allowed to invoke or target.
- **File permissions**: permissions on the config file, state directory, log
  files and card files, and whether any of them expose data to other local users.

Denial-of-service reports against a local, single-user CLI tool (e.g. crashing
your own process with malformed local input) are lower priority than the areas
above, but are still welcome.
