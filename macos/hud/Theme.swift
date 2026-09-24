import AppKit

extension NSColor {
    convenience init(hex: String, alpha: CGFloat = 1) {
        var h = hex.trimmingCharacters(in: .whitespaces)
        if h.hasPrefix("#") { h.removeFirst() }
        var v: UInt64 = 0
        Scanner(string: h).scanHexInt64(&v)
        self.init(srgbRed: CGFloat((v >> 16) & 0xFF) / 255, green: CGFloat((v >> 8) & 0xFF) / 255,
                  blue: CGFloat(v & 0xFF) / 255, alpha: alpha)
    }

    /// Resolves to `dark` or `light` from whichever appearance the view/window is actually drawing in.
    static func dynamic(dark: String, light: String, alpha: CGFloat = 1) -> NSColor {
        NSColor(name: nil) { appearance in
            let isDark = appearance.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua
            return NSColor(hex: isDark ? dark : light, alpha: alpha)
        }
    }
}

/// Claude's own palette: one terracotta accent, no per-event colour coding.
enum Theme {
    static let accent = NSColor(hex: "D97757")
    static let accentPressed = NSColor(hex: "C6613F")
    static let contextDanger = NSColor(hex: "BF4D43")

    static let background = NSColor.dynamic(dark: "262624", light: "FAF9F5", alpha: 0.97)
    /// Dividers, pill fills and other subtle surfaces; light theme reuses its one border tone.
    static let surface = NSColor.dynamic(dark: "30302E", light: "F0EEE6")
    static let border = NSColor.dynamic(dark: "3A3936", light: "E8E6DC")
    static let textPrimary = NSColor.dynamic(dark: "F5F4EE", light: "141413")
    static let textSecondary = NSColor.dynamic(dark: "A6A39A", light: "5E5D59")
    static let textTertiary = NSColor.dynamic(dark: "77756E", light: "87867F")

    private static let appIconPath = "/Applications/Claude.app/Contents/Resources/electron.icns"

    /// Claude's real app icon, used as-is (no rounding) when the app is installed.
    static func appIcon(size: CGFloat) -> NSImage? {
        guard FileManager.default.fileExists(atPath: appIconPath), let image = NSImage(contentsOfFile: appIconPath) else { return nil }
        image.size = NSSize(width: size, height: size)
        return image
    }

    /// Fallback mark when Claude.app isn't installed: a hand-drawn-looking starburst radiating from the centre.
    static func sparkIcon(size: CGFloat, color: NSColor) -> NSImage {
        let image = NSImage(size: NSSize(width: size, height: size))
        image.lockFocus()
        let center = NSPoint(x: size / 2, y: size / 2)
        // Alternating long/short rays with slight irregularity read as hand-drawn, not a mechanical asterisk.
        let lengthFactors: [CGFloat] = [1.0, 0.5, 0.92, 0.46, 1.05, 0.52, 0.88, 0.44, 0.98, 0.5]
        let innerRadius = size * 0.08
        let path = NSBezierPath()
        path.lineWidth = size * 0.085
        path.lineCapStyle = .round
        for (i, factor) in lengthFactors.enumerated() {
            let angle = (CGFloat(i) / CGFloat(lengthFactors.count)) * 2 * .pi
            let length = size * 0.46 * factor
            path.move(to: NSPoint(x: center.x + cos(angle) * innerRadius, y: center.y + sin(angle) * innerRadius))
            path.line(to: NSPoint(x: center.x + cos(angle) * length, y: center.y + sin(angle) * length))
        }
        color.setStroke()
        path.stroke()
        image.unlockFocus()
        return image
    }

    static func symbolImage(_ name: String, pointSize: CGFloat, weight: NSFont.Weight = .regular, color: NSColor) -> NSImage? {
        guard let base = NSImage(systemSymbolName: name, accessibilityDescription: nil) else { return nil }
        let sizeConfig = NSImage.SymbolConfiguration(pointSize: pointSize, weight: weight)
        let colorConfig = NSImage.SymbolConfiguration(paletteColors: [color])
        return base.withSymbolConfiguration(sizeConfig.applying(colorConfig))
    }

    /// Inline glyph + trailing space, baseline-nudged to sit with the font's cap height.
    static func attachmentString(_ symbolName: String, font: NSFont, color: NSColor) -> NSAttributedString {
        let glyphSize = round(font.pointSize * 0.92)
        guard let image = symbolImage(symbolName, pointSize: glyphSize, weight: .medium, color: color) else {
            return NSAttributedString(string: "")
        }
        let attachment = NSTextAttachment()
        attachment.image = image
        let imageSize = image.size
        let yOffset = (font.capHeight - imageSize.height) / 2
        attachment.bounds = CGRect(x: 0, y: yOffset, width: imageSize.width, height: imageSize.height)
        return NSAttributedString(attachment: attachment)
    }

    /// Neutral fill except near the ceiling, where it turns into the one warning colour the palette allows.
    static func meterColor(fraction: Double) -> NSColor { fraction >= 0.85 ? contextDanger : accent }
}

/// Every pixel dimension the card uses, derived once from the screen-relative `scale` factor.
struct Metrics {
    let scale: CGFloat

    var pad: CGFloat { round(14 * scale) }
    /// Clamped to the 28-32pt band the spec calls for regardless of screen scale.
    var iconSize: CGFloat { min(32, max(28, round(30 * scale))) }
    var headerGap: CGFloat { round(10 * scale) }
    var titleFontSize: CGFloat { round(13 * scale) }
    var timeFontSize: CGFloat { round(11 * scale) }
    var closeButtonSize: CGFloat { round(18 * scale) }
    var titleGap: CGFloat { round(8 * scale) }
    var rowGap: CGFloat { round(3 * scale) }
    var projectFontSize: CGFloat { round(12 * scale) }
    var remoteFontSize: CGFloat { 10.5 * scale }
    var bodyFontSize: CGFloat { round(12 * scale) }
    var bodyGap: CGFloat { round(8 * scale) }
    var pillFontSize: CGFloat { 10.5 * scale }
    var pillHeight: CGFloat { round(20 * scale) }
    var pillHPad: CGFloat { round(8 * scale) }
    var pillGap: CGFloat { round(6 * scale) }
    var pillsGapAbove: CGFloat { round(8 * scale) }
    var dividerGapTop: CGFloat { round(10 * scale) }
    var dividerGapBottom: CGFloat { round(9 * scale) }
    var footerFontSize: CGFloat { 10.5 * scale }
    var footerItemGap: CGFloat { round(12 * scale) }
    var meterWidth: CGFloat { round(50 * scale) }
    var meterHeight: CGFloat { 4 }
    var focusButtonHeight: CGFloat { round(22 * scale) }
    var countdownHeight: CGFloat { 3 }
    var bottomPad: CGFloat { round(12 * scale) }
}
