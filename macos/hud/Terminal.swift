import AppKit
import Darwin

/// Runs a foreground process, killing it if it overruns; nil on any failure so callers degrade gracefully.
enum Shell {
    static func run(_ executable: String, _ args: [String], timeout: TimeInterval = 2) -> String? {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = args
        let stdout = Pipe()
        let stderr = Pipe()
        process.standardOutput = stdout
        process.standardError = stderr
        process.standardInput = FileHandle.nullDevice

        do {
            try process.run()
        } catch {
            return nil
        }

        let deadline = Date().addingTimeInterval(timeout)
        while process.isRunning && Date() < deadline {
            usleep(10_000)
        }
        if process.isRunning {
            process.terminate()
            usleep(20_000)
            if process.isRunning { kill(process.processIdentifier, SIGKILL) }
            return nil
        }

        guard process.terminationStatus == 0 else { return nil }
        let data = stdout.fileHandleForReading.readDataToEndOfFile()
        guard let text = String(data: data, encoding: .utf8) else { return nil }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

/// Brings the terminal named by a card's FocusTarget to the front; detection itself lives in the Go core.
enum Terminal {
    /// Only [A-Za-z0-9-/] is allowed through into an AppleScript source string.
    private static func sanitized(_ s: String) -> String? {
        let allowed = CharacterSet(charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-/")
        guard !s.isEmpty, s.unicodeScalars.allSatisfy({ allowed.contains($0) }) else { return nil }
        return s
    }

    static func focus(_ target: CardFocusTarget) {
        if let socket = target.tmuxSocket, let pane = target.tmuxPane, !socket.isEmpty, !pane.isEmpty {
            _ = Shell.run("/usr/bin/tmux", ["-S", socket, "select-window", "-t", pane], timeout: 2)
            _ = Shell.run("/usr/bin/tmux", ["-S", socket, "select-pane", "-t", pane], timeout: 2)
        }

        switch target.terminal {
        case "iTerm.app":
            guard let raw = target.itermSessionId, let sessionId = sanitized(raw) else { return }
            let script = """
            tell application "iTerm2"
                activate
                repeat with w in windows
                    repeat with t in tabs of w
                        repeat with s in sessions of t
                            if (id of s) is "\(sessionId)" then
                                select w
                                select t
                                select s
                                return
                            end if
                        end repeat
                    end repeat
                end repeat
            end tell
            """
            _ = Shell.run("/usr/bin/osascript", ["-e", script], timeout: 2)
        case "Apple_Terminal":
            guard let raw = target.tty, let tty = sanitized(stripDevPrefix(raw)) else {
                _ = Shell.run("/usr/bin/open", ["-a", "Terminal"], timeout: 2)
                return
            }
            let script = """
            tell application "Terminal"
                activate
                repeat with w in windows
                    repeat with tb in tabs of w
                        if tty of tb is "/dev/\(tty)" then
                            set selected tab of w to tb
                            set index of w to 1
                            return
                        end if
                    end repeat
                end repeat
            end tell
            """
            _ = Shell.run("/usr/bin/osascript", ["-e", script], timeout: 2)
        case "vscode":
            let appName = target.bundleId == "com.todesktop.230313mzl4w4u92" ? "Cursor" : "Visual Studio Code"
            _ = Shell.run("/usr/bin/open", ["-a", appName, target.cwd], timeout: 2)
        case "WezTerm":
            if let pane = target.weztermPane, !pane.isEmpty {
                _ = Shell.run("/usr/bin/env", ["wezterm", "cli", "activate-pane", "--pane-id", pane], timeout: 2)
            }
            _ = Shell.run("/usr/bin/open", ["-a", "WezTerm"], timeout: 2)
        case "kitty":
            if let listenOn = target.kittyListenOn, let windowId = target.kittyWindowId, !listenOn.isEmpty {
                _ = Shell.run("/usr/bin/env", ["kitty", "@", "--to", listenOn, "focus-window", "--match", "id:\(windowId)"], timeout: 2)
            } else if let bundleId = target.bundleId {
                activateApp(bundleId: bundleId)
            }
        default:
            if let bundleId = target.bundleId {
                activateApp(bundleId: bundleId)
            }
        }
    }

    /// AppleScript screen coordinates (top-left origin) for placing the HUD on the right monitor; only iTerm2/Terminal expose bounds this way.
    static func windowCenter(_ target: CardFocusTarget) -> CGPoint? {
        switch target.terminal {
        case "iTerm.app":
            guard let raw = target.itermSessionId, let sessionId = sanitized(raw) else { return nil }
            let script = """
            tell application "iTerm2"
                repeat with w in windows
                    repeat with t in tabs of w
                        repeat with s in sessions of t
                            if (id of s) is "\(sessionId)" then
                                set b to bounds of w
                                return ((item 1 of b) as string) & "," & ((item 2 of b) as string) & "," & ((item 3 of b) as string) & "," & ((item 4 of b) as string)
                            end if
                        end repeat
                    end repeat
                end repeat
                return ""
            end tell
            """
            return centerFromBoundsScript(script)
        case "Apple_Terminal":
            guard let raw = target.tty, let tty = sanitized(stripDevPrefix(raw)) else { return nil }
            let script = """
            tell application "Terminal"
                repeat with w in windows
                    repeat with tb in tabs of w
                        if tty of tb is "/dev/\(tty)" then
                            set b to bounds of w
                            return ((item 1 of b) as string) & "," & ((item 2 of b) as string) & "," & ((item 3 of b) as string) & "," & ((item 4 of b) as string)
                        end if
                    end repeat
                end repeat
                return ""
            end tell
            """
            return centerFromBoundsScript(script)
        default:
            return nil
        }
    }

    /// True only when the exact tab/pane can be verified, never just "some window of this app".
    static func isFocused(_ target: CardFocusTarget) -> Bool {
        guard let bundleId = target.bundleId,
              let frontmost = NSWorkspace.shared.frontmostApplication?.bundleIdentifier,
              frontmost == bundleId else { return false }

        if let socket = target.tmuxSocket, let pane = target.tmuxPane, !socket.isEmpty, !pane.isEmpty {
            guard let active = Shell.run("/usr/bin/tmux", ["-S", socket, "display-message", "-p", "#{pane_id}"], timeout: 1),
                  active == pane else { return false }
        }

        switch target.terminal {
        case "iTerm.app":
            guard let raw = target.itermSessionId, let sessionId = sanitized(raw) else { return false }
            let script = """
            tell application "iTerm2"
                try
                    return id of current session of current window is "\(sessionId)"
                on error
                    return false
                end try
            end tell
            """
            return Shell.run("/usr/bin/osascript", ["-e", script], timeout: 1) == "true"
        case "Apple_Terminal":
            guard let raw = target.tty, let tty = sanitized(stripDevPrefix(raw)) else { return false }
            let script = """
            tell application "Terminal"
                try
                    return tty of selected tab of front window is "/dev/\(tty)"
                on error
                    return false
                end try
            end tell
            """
            return Shell.run("/usr/bin/osascript", ["-e", script], timeout: 1) == "true"
        default:
            return false
        }
    }

    private static func stripDevPrefix(_ tty: String) -> String {
        tty.hasPrefix("/dev/") ? String(tty.dropFirst("/dev/".count)) : tty
    }

    private static func centerFromBoundsScript(_ script: String) -> CGPoint? {
        guard let out = Shell.run("/usr/bin/osascript", ["-e", script], timeout: 2) else { return nil }
        let parts = out.split(separator: ",").compactMap { Double($0.trimmingCharacters(in: .whitespaces)) }
        guard parts.count == 4 else { return nil }
        return CGPoint(x: (parts[0] + parts[2]) / 2, y: (parts[1] + parts[3]) / 2)
    }

    private static func activateApp(bundleId: String) {
        if let app = NSRunningApplication.runningApplications(withBundleIdentifier: bundleId).first {
            app.activate(options: [.activateIgnoringOtherApps])
        } else {
            _ = Shell.run("/usr/bin/open", ["-b", bundleId], timeout: 2)
        }
    }
}
