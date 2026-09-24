---
description: Change cla-notify settings (text, sound, duration, corner, theme, events) or show them
argument-hint: "[what to change, in plain words]"
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/bin/cla-notify:*)
---

The user describes a change in plain words ($ARGUMENTS), such as "turn off the done sound",
"move cards to the bottom left", "make the ask title say something else", or "use the dark
theme". Translate that into one or more `cla-notify config` calls with the Bash tool, using dot
paths. Every key below is under `"${CLAUDE_PLUGIN_ROOT}/bin/cla-notify" config get|set|unset <key>
[value]`. `set` parses booleans and numbers on its own; string values with `{placeholders}` need
no quoting beyond the shell's own.

If no arguments were given, run `"${CLAUDE_PLUGIN_ROOT}/bin/cla-notify" config show` and summarize
the non-default settings for the user.

## Keys

- `enabled` (bool) - turn all notifications on or off.
- `events.ask.enabled` / `events.permission.enabled` / `events.idle.enabled` / `events.done.enabled`
  (bool) - turn one event on or off.
- `events.<ask|permission|idle|done>.title` / `.body` (string, template) - the notification text.
- `events.<ask|permission|idle|done>.sound` (string) - `"default"` for the platform sound,
  `"none"` or `""` for silent, or a file path (`~` expands) or platform sound name.
- `events.<ask|permission|idle|done>.durationSeconds` (number, 0 or more) - how long the card stays
  up, in seconds; 0 means it stays until dismissed.
- `events.done.minTurnSeconds` (number, 0 or more) - skip the done notification for turns shorter
  than this.
- `display.corner` (string) - `top-right`, `top-left`, `bottom-right` or `bottom-left`.
- `display.screen` (string) - `auto`, `main` or `mouse`.
- `display.theme` (string) - `auto`, `light` or `dark`.
- `display.respectDock` (bool) - keep cards clear of the macOS dock.
- `display.maxCards` (number, 1-10) - how many cards can stack before the oldest closes.
- `display.suppressWhenFocused` (bool) - skip the notification when the terminal is already frontmost.
- `display.showOptions` / `.showContext` / `.showModel` / `.showGit` (bool) - card detail toggles.
- `sound.volume` (number, 0-1) - notification sound volume.
- `labels.ask` / `.wait` / `.done` (string) - the small kind label on a card.
- `labels.focusButton` (string, template) - the button that brings the terminal to front.
- `labels.dismiss` (string) - the dismiss button text.
- `labels.moreQuestions` (string, template) - the "+N more" label when a question has several options.
- `contextWindow.default` (number, greater than 0) - context window size assumed for an unlisted model.
- `contextWindow.models.<model>` (number, greater than 0) - context window size for a named model,
  e.g. `contextWindow.models.opus`.

## Template placeholders

`{project}` `{path}` `{branch}` `{repo}` `{model}` `{context}` `{contextPercent}` `{duration}`
`{question}` `{options}` `{message}` `{summary}` `{app}` `{event}` `{count}` (`labels.moreQuestions`
only). Unknown placeholders are left as written.

After any change, run `"${CLAUDE_PLUGIN_ROOT}/bin/cla-notify" config validate` and report problems,
if any, plus the key(s) that were changed.
