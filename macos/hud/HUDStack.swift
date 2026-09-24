import AppKit

enum Paths {
    static let home = FileManager.default.homeDirectoryForCurrentUser

    static var stateDir: URL {
        let dir: URL
        if let custom = ProcessInfo.processInfo.environment["CLA_NOTIFY_STATE_DIR"], !custom.isEmpty {
            dir = URL(fileURLWithPath: (custom as NSString).expandingTildeInPath)
        } else {
            dir = home.appendingPathComponent("Library/Application Support/cla-notify")
        }
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true,
                                                   attributes: [.posixPermissions: 0o700])
        return dir
    }
}

extension NSScreen {
    /// CGDirectDisplayID for this screen; 0 (never a real display) if the key is somehow missing.
    var displayID: CGDirectDisplayID {
        (deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? NSNumber)?.uint32Value ?? 0
    }
}

/// One HUD process's on-disk footprint, so concurrent `cla-notify-hud show` processes can stack without colliding.
struct StackEntry: Codable {
    var pid: Int32
    /// Process start time; a recycled pid has a different one, so it is never mistaken for a live card.
    var startedAt: UInt64
    var sessionId: String
    var createdAt: Double
    var height: Double
    var screenId: CGDirectDisplayID
    var corner: String
}

/// File-based coordination for simultaneously visible cards: the oldest sits in the corner, each newer one stacks on top.
enum HUDStack {
    static let gap: Double = 10

    private static var dir: URL {
        let d = Paths.stateDir.appendingPathComponent("hud")
        try? FileManager.default.createDirectory(at: d, withIntermediateDirectories: true,
                                                   attributes: [.posixPermissions: 0o700])
        return d
    }

    private static func fileURL(pid: Int32) -> URL { dir.appendingPathComponent("\(pid).json") }
    private static var mypid: Int32 { ProcessInfo.processInfo.processIdentifier }

    /// Live entries; stale files for processes that no longer exist are swept as a side effect.
    static func liveEntries() -> [StackEntry] {
        guard let files = try? FileManager.default.contentsOfDirectory(at: dir, includingPropertiesForKeys: nil) else { return [] }
        var result: [StackEntry] = []
        for file in files where file.pathExtension == "json" {
            guard let data = try? Data(contentsOf: file),
                  let entry = try? JSONDecoder().decode(StackEntry.self, from: data) else {
                try? FileManager.default.removeItem(at: file)
                continue
            }
            if entry.startedAt != 0, processStart(entry.pid) == entry.startedAt {
                result.append(entry)
            } else {
                try? FileManager.default.removeItem(at: file)
            }
        }
        return result
    }

    /// Registers this card, closing any card from the same session and any beyond `maxStack` (oldest first).
    static func register(_ entry: StackEntry, maxStack: Int) {
        let others = liveEntries().filter { $0.pid != entry.pid }
        var victims = others.filter { $0.sessionId == entry.sessionId }
        let survivors = others.filter { $0.sessionId != entry.sessionId }.sorted { $0.createdAt < $1.createdAt }
        victims += survivors.prefix(max(0, survivors.count - (maxStack - 1)))
        for victim in victims {
            try? FileManager.default.removeItem(at: fileURL(pid: victim.pid))
            kill(victim.pid, SIGTERM)
        }
        write(entry)
        notifyPeers()
    }

    /// Start time in microseconds since the epoch, or nil if `pid` is not running.
    static func processStart(_ pid: Int32) -> UInt64? {
        var info = proc_bsdinfo()
        let size = Int32(MemoryLayout<proc_bsdinfo>.size)
        guard proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, size) == size else { return nil }
        return info.pbi_start_tvsec * 1_000_000 + info.pbi_start_tvusec
    }

    static func write(_ entry: StackEntry) {
        guard let data = try? JSONEncoder().encode(entry) else { return }
        let url = fileURL(pid: entry.pid)
        try? data.write(to: url, options: .atomic)
        try? FileManager.default.setAttributes([.posixPermissions: 0o600], ofItemAtPath: url.path)
    }

    static func unregister() {
        try? FileManager.default.removeItem(at: fileURL(pid: mypid))
        notifyPeers()
    }

    /// Asks every other card to recompute its position now instead of waiting for its fallback timer.
    static func notifyPeers() {
        for entry in liveEntries() where entry.pid != mypid { kill(entry.pid, SIGUSR1) }
    }

    /// Distance from the corner: the heights of every older card on the same screen and corner, plus gaps.
    static func offset(for me: StackEntry) -> Double {
        liveEntries()
            .filter { $0.pid != me.pid && $0.screenId == me.screenId && $0.corner == me.corner }
            .filter { ($0.createdAt, $0.pid) < (me.createdAt, me.pid) }
            .reduce(0) { $0 + $1.height + gap }
    }
}
