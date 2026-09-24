# Changelog

## 0.0.2 - 2026-09-24

Fixes found by running the plugin end to end on every platform.

- macOS: cards now appear on macOS 15 and older; the card timestamp format was rejected
  there, so nothing was shown.
- macOS: clicking a card returns to the right iTerm2 or Terminal tab again, the card stays
  quiet while that tab is in front, and it opens on the terminal's screen. AppleScript now
  addresses both apps by bundle id, since current iTerm2 builds are named "iTerm".
- `cla-notify focus` works on macOS.

## 0.0.1 - 2026-09-24

First public release.

- Desktop notification when Claude asks a question, waits for approval or input, or
  finishes a turn, on macOS, Linux and Windows.
- macOS: native cards drawn by the bundled HUD, stacked, with options, model, context
  usage and git branch. Linux: freedesktop notifications over D-Bus. Windows: toast
  notifications, no admin rights needed.
- Clicking a card returns to the terminal the session runs in: iTerm2, Terminal, VS Code,
  Cursor, WezTerm, kitty, Ghostty, Warp, tmux, Windows Terminal and more.
- Every text, sound, duration, corner, theme and event can be changed with
  `cla-notify config` or `/cla-notify:config`, with validation of every value.
- `/cla-notify:test` shows sample cards and `/cla-notify:doctor` checks the setup.
