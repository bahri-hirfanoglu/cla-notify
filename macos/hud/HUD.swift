import AppKit
import Dispatch

/// Entry point: load the card, honour the focused-quiet rule, then hand off to the app run loop.
enum HUD {
    static func run(cardPath: String) {
        guard let card = CardDocument.load(path: cardPath) else { exit(1) }
        try? FileManager.default.removeItem(atPath: cardPath)
        if card.display.suppressWhenFocused && Terminal.isFocused(card.focus) { return }

        let app = NSApplication.shared
        app.setActivationPolicy(.accessory)
        let controller = HUDController(card: card)
        app.delegate = controller
        app.run()
    }

    /// Brings the card's terminal to the front without drawing anything; used by `cla-notify focus`.
    static func focus(cardPath: String) {
        guard let card = CardDocument.load(path: cardPath) else { exit(1) }
        Terminal.focus(card.focus)
    }
}

/// Card.Display carries no width/margin/radius/opacity/bodyLines: these are fixed layout constants, not settings.
private enum Layout {
    static let margin: CGFloat = 30
    static let opacity: Double = 1.0
    static let bodyLines = 4
}

/// Owns the panel for a single card: geometry, stacking, animation, timers and the SIGTERM eviction contract.
final class HUDController: NSObject, NSApplicationDelegate {
    private let card: CardDocument
    private let debug = ProcessInfo.processInfo.environment["CLA_NOTIFY_DEBUG"] == "1"

    private var panel: NSPanel?
    private var cardView: CardView?
    private var countdownTimer: Timer?
    private var restackTimer: Timer?
    private var signalSources: [DispatchSourceSignal] = []
    private var stackEntry: StackEntry?
    private var stackTargetY: CGFloat?
    private var isPaused = false
    private var remainingSeconds: Double
    private let totalSeconds: Double
    private let isSticky: Bool
    private var isFinishing = false
    private var corner = "bottom-right"
    private var screenId: CGDirectDisplayID = 0

    init(card: CardDocument) {
        self.card = card
        self.isSticky = card.durationSeconds <= 0
        self.totalSeconds = max(1, card.durationSeconds)
        self.remainingSeconds = self.totalSeconds
        super.init()
    }

    private func log(_ s: @autoclosure () -> String) {
        guard debug, let data = (s() + "\n").data(using: .utf8) else { return }
        FileHandle.standardError.write(data)
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        let screen = pickScreen()
        screenId = screen.displayID
        let area = card.display.respectDock ? screen.visibleFrame : screen.frame
        corner = card.display.corner.lowercased()

        let scale = clamp(area.width / 1512.0, 0.92, 1.30)
        let width = clamp(round(area.width * 0.22), 340, 420)
        let margin = Layout.margin
        let radius = round(14 * scale)
        let maxStack = max(1, card.display.maxCards)
        let appearance = themeAppearance() ?? NSApp.effectiveAppearance

        let model = CardModel(card: card, width: width, metrics: Metrics(scale: scale), bodyLines: Layout.bodyLines,
                               showOptions: card.display.showOptions, showModel: card.display.showModel,
                               showContext: card.display.showContext, showGit: card.display.showGit, isSticky: isSticky)
        let cardView = CardView(model: model, appearance: appearance)
        let onRight = !corner.contains("left")
        cardView.exitDirection = onRight ? 1 : -1
        self.cardView = cardView

        let mypid = ProcessInfo.processInfo.processIdentifier
        let entry = StackEntry(pid: mypid, startedAt: HUDStack.processStart(mypid) ?? 0, sessionId: card.sessionId,
                               createdAt: Date().timeIntervalSince1970, height: Double(cardView.contentHeight),
                               screenId: screenId, corner: corner)
        stackEntry = entry
        installSignalHandlers(marginY: margin, area: area)
        HUDStack.register(entry, maxStack: maxStack)

        let finalX = onRight ? (area.maxX - width - margin) : (area.minX + margin)
        let finalY = stackedY(marginY: margin, area: area)
        stackTargetY = finalY

        log("screen=\(screen.frame) area=\(area) scale=\(scale) width=\(width) height=\(cardView.contentHeight) corner=\(corner) pos=(\(finalX),\(finalY))")

        let panel = NSPanel(contentRect: NSRect(x: finalX, y: finalY, width: width, height: cardView.contentHeight),
                             styleMask: [.borderless, .nonactivatingPanel], backing: .buffered, defer: false)
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.level = .statusBar
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary]
        panel.isMovableByWindowBackground = false
        panel.appearance = appearance
        cardView.layer?.cornerRadius = radius
        panel.contentView = cardView
        self.panel = panel

        cardView.onFocus = { [weak self] in self?.handleFocusTapped() }
        cardView.onClose = { [weak self] in self?.dismiss(slide: true) }
        cardView.onSwipeDismiss = { [weak self] in self?.dismiss(slide: true) }
        cardView.onHoverChanged = { [weak self] hovering in
            guard let self = self else { return }
            self.isPaused = hovering
            // Never let a card vanish the instant the mouse leaves it: give the user at least 1.5s to react.
            if !hovering, !self.isSticky, self.remainingSeconds < 1.5 { self.remainingSeconds = 1.5 }
        }

        presentPanel(finalX: finalX, finalY: finalY, opacity: Layout.opacity, onRight: onRight)
        playSoundIfNeeded()
        if !isSticky { startCountdown() }
        startRestackTimer(marginY: margin, area: area)
    }

    // MARK: Screen selection

    private func pickScreen() -> NSScreen {
        let screens = NSScreen.screens
        let want = card.display.screen.lowercased()
        if let idx = Int(want), idx >= 0, idx < screens.count { return screens[idx] }
        if want == "main", let main = NSScreen.main { return main }
        if want != "mouse", let raw = Terminal.windowCenter(card.focus), let p = cocoaPoint(fromTopLeftFlipped: raw),
           let match = screens.first(where: { NSMouseInRect(p, $0.frame, false) }) {
            log("anchor point \(p) -> screen \(match.frame)")
            return match
        }
        let mouse = NSEvent.mouseLocation
        if let match = screens.first(where: { NSMouseInRect(mouse, $0.frame, false) }) { return match }
        return NSScreen.main ?? screens[0]
    }

    /// AppleScript screen coordinates are top-left origin, y-down; Cocoa is bottom-left origin, y-up.
    private func cocoaPoint(fromTopLeftFlipped p: CGPoint) -> NSPoint? {
        guard let primary = NSScreen.screens.first else { return nil }
        return NSPoint(x: p.x, y: primary.frame.maxY - p.y)
    }

    private func themeAppearance() -> NSAppearance? {
        switch card.display.theme.lowercased() {
        case "dark": return NSAppearance(named: .darkAqua)
        case "light": return NSAppearance(named: .aqua)
        default: return nil // follow the system
        }
    }

    // MARK: Presentation

    private func presentPanel(finalX: CGFloat, finalY: CGFloat, opacity: Double, onRight: Bool) {
        guard let panel = panel else { return }
        let reduceMotion = NSWorkspace.shared.accessibilityDisplayShouldReduceMotion
        let slide: CGFloat = reduceMotion ? 0 : (onRight ? 24 : -24)
        panel.setFrameOrigin(NSPoint(x: finalX + slide, y: finalY))
        panel.alphaValue = 0
        panel.orderFrontRegardless()
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.22
            ctx.timingFunction = CAMediaTimingFunction(name: .easeOut)
            panel.animator().setFrameOrigin(NSPoint(x: finalX, y: finalY))
            panel.animator().alphaValue = opacity
        }
    }

    private func playSoundIfNeeded() {
        guard let path = card.sound, let sound = NSSound(contentsOfFile: path, byReference: true) else { return }
        sound.volume = Float(max(0, min(1, card.soundVolume)))
        sound.play()
    }

    // MARK: Countdown

    private func startCountdown() {
        let interval = 0.05
        countdownTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] _ in
            guard let self = self, !self.isPaused else { return }
            self.remainingSeconds -= interval
            self.cardView?.setCountdownFraction(CGFloat(max(0, self.remainingSeconds / self.totalSeconds)))
            if self.remainingSeconds <= 0 { self.dismiss(slide: true) }
        }
    }

    // MARK: Restacking

    private func stackedY(marginY: CGFloat, area: NSRect) -> CGFloat {
        guard let entry = stackEntry, let cardView = cardView else { return area.minY + marginY }
        let offset = CGFloat(HUDStack.offset(for: entry))
        return corner.contains("top") ? area.maxY - cardView.contentHeight - marginY - offset : area.minY + marginY + offset
    }

    /// Peers signal SIGUSR1 on every change; this slow poll only covers a peer that died without signalling.
    private func startRestackTimer(marginY: CGFloat, area: NSRect) {
        restackTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.restack(marginY: marginY, area: area)
        }
    }

    private func restack(marginY: CGFloat, area: NSRect) {
        guard let panel = panel, !isFinishing else { return }
        let targetY = stackedY(marginY: marginY, area: area)
        // Compare with the last target, not the live frame, so an in-flight slide is not restarted.
        guard abs((stackTargetY ?? panel.frame.origin.y) - targetY) > 0.5 else { return }
        stackTargetY = targetY
        log("restack \(card.sessionId) -> y=\(targetY)")
        NSAnimationContext.runAnimationGroup { ctx in
            ctx.duration = 0.22
            ctx.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
            panel.animator().setFrameOrigin(NSPoint(x: panel.frame.origin.x, y: targetY))
        }
    }

    // MARK: Actions

    private func handleFocusTapped() {
        guard !isFinishing else { return }
        isFinishing = true
        countdownTimer?.invalidate()
        let target = card.focus
        DispatchQueue.global(qos: .userInitiated).async {
            Terminal.focus(target)
            DispatchQueue.main.async { [weak self] in self?.finish(slide: false, duration: 0.12) }
        }
    }

    private func dismiss(slide: Bool) {
        guard !isFinishing else { return }
        isFinishing = true
        countdownTimer?.invalidate()
        finish(slide: slide, duration: 0.18)
    }

    private func finish(slide: Bool, duration: Double) {
        restackTimer?.invalidate()
        signalSources.forEach { $0.cancel() }
        HUDStack.unregister()
        guard let panel = panel, let cardView = cardView else { NSApp.terminate(nil); return }
        let dx = slide ? 8 * cardView.exitDirection : 0
        NSAnimationContext.runAnimationGroup({ ctx in
            ctx.duration = duration
            ctx.timingFunction = CAMediaTimingFunction(name: .easeIn)
            panel.animator().alphaValue = 0
            panel.animator().setFrameOrigin(NSPoint(x: panel.frame.origin.x + dx, y: panel.frame.origin.y))
        }, completionHandler: {
            panel.orderOut(nil)
            NSApp.terminate(nil)
        })
    }

    // MARK: Signals

    /// SIGTERM: a newer card replaced or evicted us. SIGUSR1: the stack changed, move to the new position.
    private func installSignalHandlers(marginY: CGFloat, area: NSRect) {
        signal(SIGTERM, SIG_IGN)
        signal(SIGUSR1, SIG_IGN)
        let term = DispatchSource.makeSignalSource(signal: SIGTERM, queue: .main)
        term.setEventHandler { [weak self] in self?.evictedByPeer() }
        let usr1 = DispatchSource.makeSignalSource(signal: SIGUSR1, queue: .main)
        usr1.setEventHandler { [weak self] in self?.restack(marginY: marginY, area: area) }
        [term, usr1].forEach { $0.resume() }
        signalSources = [term, usr1]
    }

    private func evictedByPeer() {
        guard !isFinishing else { return }
        isFinishing = true
        countdownTimer?.invalidate()
        restackTimer?.invalidate()
        HUDStack.unregister()
        guard let panel = panel else { NSApp.terminate(nil); return }
        NSAnimationContext.runAnimationGroup({ ctx in
            ctx.duration = 0.1
            panel.animator().alphaValue = 0
        }, completionHandler: {
            panel.orderOut(nil)
            NSApp.terminate(nil)
        })
    }

    private func clamp(_ v: CGFloat, _ lo: CGFloat, _ hi: CGFloat) -> CGFloat { min(max(v, lo), hi) }
}

/// Offscreen rendering for design review; needs no Screen Recording permission because nothing is captured from the display.
extension HUD {
    static func render(cardPath: String, outPath: String, theme: String, width: CGFloat) {
        guard let card = CardDocument.load(path: cardPath) else { exit(1) }
        _ = NSApplication.shared
        let appearance = NSAppearance(named: theme == "light" ? .aqua : .darkAqua) ?? NSApp.effectiveAppearance
        let model = CardModel(card: card, width: width, metrics: Metrics(scale: 1.0), bodyLines: Layout.bodyLines,
                              showOptions: card.display.showOptions, showModel: card.display.showModel,
                              showContext: card.display.showContext, showGit: card.display.showGit,
                              isSticky: card.durationSeconds <= 0)
        let cardView = CardView(model: model, appearance: appearance)
        cardView.layer?.cornerRadius = 14
        cardView.setCountdownFraction(0.62)

        let pad: CGFloat = 24
        let canvas = NSView(frame: NSRect(x: 0, y: 0, width: width + pad * 2, height: cardView.contentHeight + pad * 2))
        canvas.wantsLayer = true
        canvas.layer?.backgroundColor = (theme == "light" ? NSColor(white: 0.82, alpha: 1) : NSColor(white: 0.12, alpha: 1)).cgColor
        cardView.frame.origin = NSPoint(x: pad, y: pad)
        canvas.addSubview(cardView)

        let window = NSWindow(contentRect: canvas.frame, styleMask: .borderless, backing: .buffered, defer: false)
        window.appearance = appearance
        window.contentView = canvas
        window.setFrameOrigin(NSPoint(x: -10_000, y: -10_000))
        window.orderFrontRegardless()
        canvas.layoutSubtreeIfNeeded()
        canvas.displayIfNeeded()
        CATransaction.flush()
        RunLoop.current.run(until: Date().addingTimeInterval(0.4))

        let scale: CGFloat = 2
        guard let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(canvas.bounds.width * scale),
                                         pixelsHigh: Int(canvas.bounds.height * scale), bitsPerSample: 8,
                                         samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
                                         colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0),
              let ctx = NSGraphicsContext(bitmapImageRep: rep) else { exit(1) }
        rep.size = canvas.bounds.size
        NSGraphicsContext.saveGraphicsState()
        NSGraphicsContext.current = ctx
        ctx.cgContext.scaleBy(x: scale, y: scale)
        appearance.performAsCurrentDrawingAppearance {
            canvas.layer?.render(in: ctx.cgContext)
        }
        NSGraphicsContext.restoreGraphicsState()
        window.orderOut(nil)
        try? rep.representation(using: .png, properties: [:])?.write(to: URL(fileURLWithPath: outPath))
    }
}
