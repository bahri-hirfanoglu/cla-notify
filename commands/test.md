---
description: Show sample cla-notify notifications (ask, permission, idle, done)
argument-hint: "[ask|permission|idle|done|all]"
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/bin/cla-notify:*)
---

Run `"${CLAUDE_PLUGIN_ROOT}/bin/cla-notify" test <kind>` with the Bash tool, where `<kind>` is
`$ARGUMENTS` if it is one of `ask`, `permission`, `idle`, `done`, `all`, otherwise `all`. Report
its one-line output to the user. If nothing appears on screen, suggest running
`/cla-notify:doctor` to check the presenter for this platform.
