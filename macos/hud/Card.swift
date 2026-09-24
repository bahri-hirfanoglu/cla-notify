import Foundation

/// Mirrors internal/card/card.go field for field; bump supportedVersion when that file's Version changes.
enum Card {
    static let supportedVersion = 1
}

/// Drives the accent colour, glyph and sound; matches card.Kind in internal/card/card.go.
enum CardKind: String, Codable {
    case ask
    case wait
    case done
}

struct CardProject: Codable {
    var name: String
    var path: String
    var branch: String?
    var remote: String?
    var dirty: Bool?
}

struct CardContext: Codable {
    var usedTokens: Int
    var windowTokens: Int

    var fraction: Double { windowTokens > 0 ? min(1, Double(usedTokens) / Double(windowTokens)) : 0 }
}

/// What the HUD needs to bring the originating terminal back to the front; matches card.FocusTarget.
struct CardFocusTarget: Codable {
    var terminal: String
    var appName: String
    var bundleId: String?
    var itermSessionId: String?
    var tty: String?
    var tmuxPane: String?
    var tmuxSocket: String?
    var weztermPane: String?
    var kittyWindowId: String?
    var kittyListenOn: String?
    var x11WindowId: String?
    var windowsHwnd: String?
    var terminalPid: Int?
    var wtSession: String?
    var cwd: String
}

struct CardDisplay: Codable {
    var corner: String
    var screen: String
    var theme: String
    var respectDock: Bool
    var maxCards: Int
    var showOptions: Bool
    var showContext: Bool
    var showModel: Bool
    var showGit: Bool
    var suppressWhenFocused: Bool
}

/// Fixed words the HUD prints, already rendered by the core from the config; matches card.Labels.
struct CardLabels: Codable {
    var kind: String
    var focusButton: String
    var dismiss: String
    var context: String
    var elapsed: String
    var moreQuestions: String
}

struct CardDocument: Codable {
    var version: Int
    var kind: CardKind
    var event: String
    var sessionId: String
    var title: String
    var body: String
    var options: [String]
    var questionCount: Int
    var project: CardProject
    var model: String?
    var context: CardContext?
    var elapsedSeconds: Double?
    var focus: CardFocusTarget
    var sound: String?
    var soundVolume: Double
    var durationSeconds: Double
    var createdAt: Date
    var display: CardDisplay
    var labels: CardLabels
}

extension CardDocument {
    private static let decoder: JSONDecoder = {
        let d = JSONDecoder()
        // .iso8601 rejects fractional seconds before macOS 26, so both forms are parsed explicitly.
        d.dateDecodingStrategy = .custom { decoder in
            let raw = try decoder.singleValueContainer().decode(String.self)
            let withFraction = ISO8601DateFormatter()
            withFraction.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            if let date = withFraction.date(from: raw) ?? ISO8601DateFormatter().date(from: raw) { return date }
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "invalid ISO 8601 date: \(raw)"))
        }
        return d
    }()

    /// Reads and decodes a card file, refusing an unsupported version with a stderr line instead of throwing.
    static func load(path: String) -> CardDocument? {
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)) else {
            fail("cannot read card file: \(path)")
            return nil
        }
        let card: CardDocument
        do {
            card = try decoder.decode(CardDocument.self, from: data)
        } catch {
            fail("cannot decode card file: \(path): \(error)")
            return nil
        }
        guard card.version == Card.supportedVersion else {
            fail("unsupported card version \(card.version), this build understands version \(Card.supportedVersion)")
            return nil
        }
        return card
    }

    private static func fail(_ message: String) {
        FileHandle.standardError.write(Data("cla-notify-hud: \(message)\n".utf8))
    }
}
