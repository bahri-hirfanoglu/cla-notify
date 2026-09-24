---
description: cla-notify diagnostics - presenter check, config and state paths, terminal detection
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/bin/cla-notify:*)
---

!`"${CLAUDE_PLUGIN_ROOT}/bin/cla-notify" doctor`

Summarize the output above for the user in a few bullets. If the presenter check failed, name what
is missing (the macOS HUD binary, a notification daemon on Linux, or WinRT toast support on
Windows) and how to fix it. If the detected terminal is unknown, explain that clicking the
notification will only bring the app to the front, not the exact tab.
