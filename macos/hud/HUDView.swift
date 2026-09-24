import AppKit

/// Display-ready inputs for `CardView`; all Card interpretation happens before this is built.
struct CardModel {
    let card: CardDocument
    let width: CGFloat
    let metrics: Metrics
    let bodyLines: Int
    let showOptions: Bool
    let showModel: Bool
    let showContext: Bool
    let showGit: Bool
    /// `durationSeconds <= 0`: no countdown bar, no auto-dismiss timer, closes only on click/close/replace.
    let isSticky: Bool
}

/// NSTextField draws its text 2pt inside its frame on each side; layout aligns and measures against the text itself.
let labelInset: CGFloat = 2

/// Width a single-line label needs to show `label`'s whole value without truncating.
func fittedWidth(_ label: NSTextField) -> CGFloat { ceil(label.cell?.cellSize.width ?? label.intrinsicContentSize.width) }

func singleLineLabel(_ text: String, font: NSFont, color: NSColor) -> NSTextField {
    let label = NSTextField(labelWithString: text)
    label.font = font
    label.textColor = color
    label.lineBreakMode = .byTruncatingTail
    label.usesSingleLineMode = true
    label.cell?.truncatesLastVisibleLine = true
    return label
}

func singleLineLabel(_ text: NSAttributedString) -> NSTextField {
    // Attributed values ignore the cell's line break mode, so the ellipsis has to live in the string itself.
    let styled = NSMutableAttributedString(attributedString: text)
    let paragraph = NSMutableParagraphStyle()
    paragraph.lineBreakMode = .byTruncatingTail
    styled.addAttribute(.paragraphStyle, value: paragraph, range: NSRange(location: 0, length: styled.length))
    let label = NSTextField(labelWithAttributedString: styled)
    label.lineBreakMode = .byTruncatingTail
    label.usesSingleLineMode = true
    label.cell?.truncatesLastVisibleLine = true
    return label
}

/// Small rounded neutral label used for option chips and the "+N" overflow chip.
final class PillView: NSView {
    init(text: String, font: NSFont, fill: NSColor, border: NSColor, textColor: NSColor, hPad: CGFloat, height: CGFloat,
         maxWidth: CGFloat = .greatestFiniteMagnitude) {
        super.init(frame: .zero)
        wantsLayer = true
        layer?.cornerRadius = height / 2
        layer?.backgroundColor = fill.cgColor
        layer?.borderWidth = 1
        layer?.borderColor = border.cgColor

        let label = singleLineLabel(text, font: font, color: textColor)
        label.alignment = .center
        let labelWidth = min(fittedWidth(label), maxWidth - hPad * 2 + labelInset * 2)
        let width = labelWidth - labelInset * 2 + hPad * 2
        let labelHeight = ceil(label.cell?.cellSize.height ?? font.pointSize + 4)
        label.frame = NSRect(x: hPad - labelInset, y: round((height - labelHeight) / 2), width: labelWidth, height: labelHeight)
        addSubview(label)
        frame = NSRect(x: 0, y: 0, width: width, height: height)
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }
}

/// Rounded token-usage bar: track + coloured fill sized to `fraction`.
final class ContextMeterView: NSView {
    private let fill = NSView()

    init(width: CGFloat, height: CGFloat, fraction: Double, color: NSColor) {
        super.init(frame: NSRect(x: 0, y: 0, width: width, height: height))
        wantsLayer = true
        layer?.cornerRadius = height / 2
        layer?.backgroundColor = Theme.textTertiary.withAlphaComponent(0.25).cgColor

        fill.wantsLayer = true
        fill.layer?.cornerRadius = height / 2
        fill.layer?.backgroundColor = color.cgColor
        fill.frame = NSRect(x: 0, y: 0, width: round(width * CGFloat(fraction)), height: height)
        addSubview(fill)
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }
}

/// Progress bar along the card's bottom edge that drains from full width to zero over the visible duration.
final class CountdownBarView: NSView {
    private let fillLayer = CALayer()

    override init(frame: NSRect) {
        super.init(frame: frame)
        wantsLayer = true
        layer?.backgroundColor = NSColor.clear.cgColor
        fillLayer.anchorPoint = CGPoint(x: 0, y: 0.5)
        layer?.addSublayer(fillLayer)
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    func configure(color: NSColor, track: NSColor) {
        layer?.backgroundColor = track.cgColor
        fillLayer.backgroundColor = color.cgColor
        fillLayer.frame = CGRect(x: 0, y: 0, width: bounds.width, height: bounds.height)
        fillLayer.position = CGPoint(x: 0, y: bounds.height / 2)
    }

    /// `fraction` is remaining-time / total-duration, 1 (full) down to 0 (empty).
    func setFraction(_ fraction: CGFloat) {
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        fillLayer.bounds = CGRect(x: 0, y: 0, width: max(0, bounds.width * fraction), height: bounds.height)
        CATransaction.commit()
    }
}

/// Button whose accent-coloured title darkens while the mouse is down, the only "interaction colour" the palette allows.
final class AccentButton: NSButton {
    var normalAttributedTitle = NSAttributedString() {
        didSet { attributedTitle = normalAttributedTitle }
    }
    var pressedAttributedTitle = NSAttributedString()

    override func mouseDown(with event: NSEvent) {
        attributedTitle = pressedAttributedTitle
        super.mouseDown(with: event) // blocks in its own tracking loop until mouse-up
        attributedTitle = normalAttributedTitle
    }
}

/// The notification card itself: header, project line, body, options, footer and (non-sticky) countdown bar.
final class CardView: NSView {
    private let model: CardModel
    private let countdownBar = CountdownBarView()
    private let closeButton = NSButton()
    private var trackingArea: NSTrackingArea?
    private var swipeAccumulated: CGFloat = 0
    private var swipeHandled = false

    private(set) var contentHeight: CGFloat = 0

    /// +1 when the card is anchored on the right edge, -1 on the left, the direction a slide/swipe exits toward.
    var exitDirection: CGFloat = 1

    var onFocus: (() -> Void)?
    var onClose: (() -> Void)?
    var onSwipeDismiss: (() -> Void)?
    var onHoverChanged: ((Bool) -> Void)?

    /// Building the view tree inside `appearance`'s drawing context makes every dynamic Theme colour resolve
    /// against the caller's chosen theme instead of whatever the system happens to be in right now.
    init(model: CardModel, appearance: NSAppearance) {
        self.model = model
        super.init(frame: .zero)
        appearance.performAsCurrentDrawingAppearance { [self] in
            wantsLayer = true
            layer?.backgroundColor = Theme.background.cgColor
            layer?.masksToBounds = true
            layer?.borderWidth = 1
            layer?.borderColor = Theme.border.cgColor
            buildContent()
        }
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    override var isFlipped: Bool { true }

    func setCountdownFraction(_ fraction: CGFloat) { countdownBar.setFraction(fraction) }

    private func buildContent() {
        let m = model.metrics
        let width = model.width
        let contentWidth = width - m.pad * 2
        var cursorY: CGFloat = m.pad

        // Reserve the close button's slot up front so hover toggling never reflows the header.
        let closeSlotWidth = m.closeButtonSize + 6

        let titleFont = NSFont.systemFont(ofSize: m.titleFontSize, weight: .semibold)
        let timeFont = NSFont.systemFont(ofSize: m.timeFontSize, weight: .regular)
        let projectFont = NSFont.systemFont(ofSize: m.projectFontSize, weight: .medium)
        let remoteFont = NSFont.monospacedSystemFont(ofSize: m.remoteFontSize, weight: .regular)
        let bodyFont = NSFont.systemFont(ofSize: m.bodyFontSize, weight: .regular)

        let titleX = m.pad + m.iconSize + m.headerGap
        let titleH = lineHeight(titleFont)
        let projectH = lineHeight(projectFont)
        let remoteH = lineHeight(remoteFont)

        // Title row: [title (truncates)] ... [time] [close]; the time is placed first so the title never slides under it.
        let timeLabel = singleLineLabel(currentTimeString(), font: timeFont, color: Theme.textTertiary)
        timeLabel.alignment = .right
        let timeWidth = fittedWidth(timeLabel)
        let timeX = width - m.pad - closeSlotWidth - timeWidth + labelInset
        // Baseline-align the smaller time with the title instead of centring their boxes.
        let timeY = cursorY + round(titleFont.ascender - timeFont.ascender)
        timeLabel.frame = NSRect(x: timeX, y: timeY, width: timeWidth, height: lineHeight(timeFont))
        addSubview(timeLabel)

        let titleLabel = singleLineLabel(model.card.title, font: titleFont, color: Theme.textPrimary)
        let titleWidth = min(fittedWidth(titleLabel), timeX - m.titleGap - (titleX - labelInset))
        titleLabel.frame = NSRect(x: titleX - labelInset, y: cursorY, width: titleWidth, height: titleH)
        addSubview(titleLabel)

        configureCloseButton(x: width - m.pad - m.closeButtonSize, y: cursorY + round((titleH - m.closeButtonSize) / 2), size: m.closeButtonSize)

        cursorY += titleH + m.rowGap

        // Project and branch truncate independently so a long project name never hides the branch entirely.
        addProjectRow(x: titleX, y: cursorY, width: width - m.pad - titleX, font: projectFont, height: projectH)
        cursorY += projectH

        if let remote = model.card.project.remote, model.showGit {
            cursorY += 2
            let remoteLabel = singleLineLabel(remote, font: remoteFont, color: Theme.textSecondary)
            remoteLabel.frame = NSRect(x: titleX - labelInset, y: cursorY, width: width - m.pad - titleX + labelInset * 2, height: remoteH)
            addSubview(remoteLabel)
            cursorY += remoteH
        }

        // App icon, vertically centred against the whole header block just laid out.
        let headerBlockHeight = cursorY - m.pad
        let iconY = m.pad + max(0, (headerBlockHeight - m.iconSize) / 2)
        addAppIcon(x: m.pad, y: iconY, size: m.iconSize)

        // Body.
        let body = model.card.body
        if !body.isEmpty {
            cursorY += m.bodyGap
            let bodyLabel = NSTextField(wrappingLabelWithString: body)
            bodyLabel.font = bodyFont
            bodyLabel.textColor = Theme.textPrimary
            bodyLabel.maximumNumberOfLines = model.bodyLines
            bodyLabel.lineBreakMode = .byWordWrapping
            bodyLabel.cell?.truncatesLastVisibleLine = true
            let bodyWidth = contentWidth + labelInset * 2
            let measured = bodyLabel.cell?.cellSize(forBounds: NSRect(x: 0, y: 0, width: bodyWidth, height: .greatestFiniteMagnitude)).height ?? 0
            let bodyH = min(ceil(measured), lineHeight(bodyFont) * CGFloat(model.bodyLines))
            bodyLabel.frame = NSRect(x: m.pad - labelInset, y: cursorY, width: bodyWidth, height: bodyH)
            addSubview(bodyLabel)
            cursorY += bodyH
        }

        // Option pills (ask events only).
        if model.card.kind == .ask, model.showOptions, !model.card.options.isEmpty {
            cursorY += m.pillsGapAbove
            if !model.card.labels.moreQuestions.isEmpty {
                let tagFont = NSFont.systemFont(ofSize: m.footerFontSize, weight: .regular)
                let tag = singleLineLabel(model.card.labels.moreQuestions, font: tagFont, color: Theme.textTertiary)
                let tagH = lineHeight(tagFont)
                tag.frame = NSRect(x: m.pad - labelInset, y: cursorY, width: fittedWidth(tag), height: tagH)
                addSubview(tag)
                cursorY += tagH + 4
            }
            let pillsHeight = addPillsRow(y: cursorY, contentWidth: contentWidth, pad: m.pad, m: m)
            cursorY += pillsHeight
        }

        // Divider + footer.
        let hasFooter = model.showModel && model.card.model != nil
            || model.showContext && model.card.context != nil
            || model.card.kind == .done && model.card.elapsedSeconds != nil
        if hasFooter {
            cursorY += m.dividerGapTop
            addDivider(y: cursorY, width: contentWidth, pad: m.pad)
            cursorY += 1 + m.dividerGapBottom
            let footerH = addFooter(y: cursorY, contentWidth: contentWidth, m: m)
            cursorY += footerH
        }

        cursorY += m.bottomPad
        if !model.isSticky { cursorY += m.countdownHeight }
        contentHeight = cursorY

        if !model.isSticky {
            countdownBar.frame = NSRect(x: 0, y: contentHeight - m.countdownHeight, width: width, height: m.countdownHeight)
            countdownBar.configure(color: Theme.accent, track: Theme.surface)
            addSubview(countdownBar)
        }

        setFrameSize(NSSize(width: width, height: contentHeight))
        setupTracking()
    }

    // MARK: Header helpers

    private func addAppIcon(x: CGFloat, y: CGFloat, size: CGFloat) {
        let image = Theme.appIcon(size: size) ?? Theme.sparkIcon(size: size, color: Theme.accent)
        let iv = NSImageView(frame: NSRect(x: x, y: y, width: size, height: size))
        iv.image = image
        iv.imageScaling = .scaleProportionallyUpOrDown
        addSubview(iv)
    }

    private func addDivider(y: CGFloat, width: CGFloat, pad: CGFloat) {
        let divider = NSView(frame: NSRect(x: pad, y: y, width: width, height: 1))
        divider.wantsLayer = true
        divider.layer?.backgroundColor = Theme.border.cgColor
        addSubview(divider)
    }

    private func addProjectRow(x: CGFloat, y: CGFloat, width: CGFloat, font: NSFont, height: CGFloat) {
        let project = model.card.project
        let projectText = NSMutableAttributedString()
        projectText.append(Theme.attachmentString("folder.fill", font: font, color: Theme.textSecondary))
        projectText.append(NSAttributedString(string: " " + project.name, attributes: [.font: font, .foregroundColor: Theme.textPrimary]))
        let projectLabel = singleLineLabel(projectText)

        guard model.showGit, let branch = project.branch else {
            projectLabel.frame = NSRect(x: x - labelInset, y: y, width: min(fittedWidth(projectLabel), width + labelInset * 2), height: height)
            addSubview(projectLabel)
            return
        }

        let branchText = NSMutableAttributedString()
        branchText.append(NSAttributedString(string: "\u{00b7}  ", attributes: [.font: font, .foregroundColor: Theme.textTertiary]))
        branchText.append(Theme.attachmentString("arrow.triangle.branch", font: font, color: Theme.textSecondary))
        branchText.append(NSAttributedString(string: " " + branch, attributes: [.font: font, .foregroundColor: Theme.textPrimary]))
        let branchLabel = singleLineLabel(branchText)

        let dotSize: CGFloat = project.dirty == true ? 6 : 0
        let dotGap: CGFloat = dotSize > 0 ? 5 : 0
        let gap: CGFloat = 6
        let available = width - gap - dotSize - dotGap
        var projectWidth = fittedWidth(projectLabel) - labelInset * 2
        var branchWidth = fittedWidth(branchLabel) - labelInset * 2
        if projectWidth + branchWidth > available {
            // Each side keeps its natural width when it fits in half; otherwise the longer one gives way.
            let half = floor(available / 2)
            if projectWidth <= half { branchWidth = available - projectWidth }
            else if branchWidth <= half { projectWidth = available - branchWidth }
            else { projectWidth = half; branchWidth = available - half }
        }

        projectLabel.frame = NSRect(x: x - labelInset, y: y, width: projectWidth + labelInset * 2, height: height)
        branchLabel.frame = NSRect(x: x + projectWidth + gap - labelInset, y: y, width: branchWidth + labelInset * 2, height: height)
        addSubview(projectLabel)
        addSubview(branchLabel)

        if dotSize > 0 {
            let dot = NSView(frame: NSRect(x: x + projectWidth + gap + branchWidth + dotGap,
                                           y: y + round((height - dotSize) / 2), width: dotSize, height: dotSize))
            dot.wantsLayer = true
            dot.layer?.cornerRadius = dotSize / 2
            dot.layer?.backgroundColor = Theme.accent.cgColor
            dot.toolTip = "Uncommitted changes"
            addSubview(dot)
        }
    }

    private func configureCloseButton(x: CGFloat, y: CGFloat, size: CGFloat) {
        closeButton.frame = NSRect(x: x, y: y, width: size, height: size)
        closeButton.isBordered = false
        closeButton.bezelStyle = .circular
        closeButton.image = Theme.symbolImage("xmark", pointSize: size * 0.5, weight: .semibold, color: Theme.textSecondary)
        closeButton.imageScaling = .scaleProportionallyDown
        closeButton.alphaValue = 0
        closeButton.target = self
        closeButton.action = #selector(closeTapped)
        closeButton.toolTip = model.card.labels.dismiss
        closeButton.setAccessibilityLabel(model.card.labels.dismiss)
        addSubview(closeButton)
    }

    @objc private func closeTapped() { onClose?() }

    // MARK: Pills

    /// One row of option chips; whatever does not fit collapses into a trailing "+N" chip that always fits.
    private func addPillsRow(y: CGFloat, contentWidth: CGFloat, pad: CGFloat, m: Metrics) -> CGFloat {
        let font = NSFont.systemFont(ofSize: m.pillFontSize, weight: .medium)
        let options = model.card.options
        func pill(_ text: String, color: NSColor, maxWidth: CGFloat = .greatestFiniteMagnitude) -> PillView {
            PillView(text: text, font: font, fill: Theme.surface, border: Theme.border, textColor: color,
                     hPad: m.pillHPad, height: m.pillHeight, maxWidth: maxWidth)
        }
        let overflowReserve = pill("+\(options.count)", color: Theme.textTertiary).frame.width + m.pillGap
        let maxPillWidth = floor(contentWidth * 0.6)

        var placed: [PillView] = []
        var x: CGFloat = 0
        for (index, text) in options.enumerated() {
            let view = pill(text, color: Theme.textSecondary, maxWidth: maxPillWidth)
            let isLast = index == options.count - 1
            let limit = isLast ? contentWidth : contentWidth - overflowReserve
            guard x + view.frame.width <= limit else { break }
            view.frame.origin = NSPoint(x: pad + x, y: y)
            placed.append(view)
            x += view.frame.width + m.pillGap
        }
        if placed.isEmpty, let first = options.first {
            let view = pill(first, color: Theme.textSecondary, maxWidth: contentWidth - overflowReserve)
            view.frame.origin = NSPoint(x: pad, y: y)
            placed.append(view)
            x = view.frame.width + m.pillGap
        }
        placed.forEach(addSubview)
        let hidden = options.count - placed.count
        if hidden > 0 {
            // Purely numeric, so no language-specific overflow word is needed here.
            let more = pill("+\(hidden)", color: Theme.textTertiary)
            more.frame.origin = NSPoint(x: pad + x, y: y)
            addSubview(more)
        }
        return m.pillHeight
    }

    // MARK: Footer

    /// Left group priority is context > model > elapsed: dropped from the low end until it clears the focus button.
    private func addFooter(y: CGFloat, contentWidth: CGFloat, m: Metrics) -> CGFloat {
        let focusFont = NSFont.systemFont(ofSize: m.footerFontSize, weight: .semibold)
        let focusTitle = model.card.labels.focusButton
        let focusNormal = footerButtonAttributedString(title: focusTitle, font: focusFont, color: Theme.accent)
        let focusPressed = footerButtonAttributedString(title: focusTitle, font: focusFont, color: Theme.accentPressed)
        let focusWidth = ceil(focusNormal.size().width) + 12
        let leftBudget = max(0, contentWidth - focusWidth - m.footerItemGap)

        let font = NSFont.systemFont(ofSize: m.footerFontSize, weight: .regular)
        let itemY = y + (m.focusButtonHeight - lineHeight(font)) / 2

        var showModel = model.showModel && model.card.model != nil
        var showElapsed = model.card.kind == .done && model.card.elapsedSeconds != nil
        let showContext = model.showContext && model.card.context != nil

        func leftWidth() -> CGFloat {
            var width: CGFloat = 0
            var count = 0
            if showModel, let name = model.card.model { width += chipWidth(symbol: "sparkles", text: name, font: font); count += 1 }
            if showContext {
                width += m.meterWidth + 5 + labelWidth(model.card.labels.context, font: font)
                count += 1
            }
            if showElapsed { width += chipWidth(symbol: "timer", text: model.card.labels.elapsed, font: font); count += 1 }
            return count > 1 ? width + CGFloat(count - 1) * m.footerItemGap : width
        }
        if leftWidth() > leftBudget { showElapsed = false }
        if leftWidth() > leftBudget { showModel = false }

        var x = m.pad
        if showModel, let name = model.card.model {
            x += addFooterChip(symbol: "sparkles", text: name, x: x, y: itemY, font: font)
            x += m.footerItemGap
        }
        if showContext, let context = model.card.context {
            let meter = ContextMeterView(width: m.meterWidth, height: m.meterHeight, fraction: context.fraction,
                                          color: Theme.meterColor(fraction: context.fraction))
            meter.frame.origin = NSPoint(x: x, y: y + (m.focusButtonHeight - m.meterHeight) / 2)
            addSubview(meter)
            x += m.meterWidth + 5
            x += addFooterLabel(model.card.labels.context, x: x, y: itemY, font: font)
            x += m.footerItemGap
        }
        if showElapsed {
            x += addFooterChip(symbol: "timer", text: model.card.labels.elapsed, x: x, y: itemY, font: font)
        }

        let button = AccentButton(frame: NSRect(x: model.width - m.pad - focusWidth, y: y, width: focusWidth, height: m.focusButtonHeight))
        button.isBordered = false
        button.normalAttributedTitle = focusNormal
        button.pressedAttributedTitle = focusPressed
        button.toolTip = model.card.focus.appName
        button.target = self
        button.action = #selector(focusTapped)
        addSubview(button)

        return m.focusButtonHeight
    }

    private func footerButtonAttributedString(title: String, font: NSFont, color: NSColor) -> NSAttributedString {
        let attr = NSMutableAttributedString()
        attr.append(Theme.attachmentString("arrow.uturn.backward", font: font, color: color))
        attr.append(NSAttributedString(string: " " + title, attributes: [.font: font, .foregroundColor: color]))
        return attr
    }

    private func footerChipString(symbol: String, text: String, font: NSFont) -> NSAttributedString {
        let attr = NSMutableAttributedString()
        attr.append(Theme.attachmentString(symbol, font: font, color: Theme.textSecondary))
        attr.append(NSAttributedString(string: " " + text, attributes: [.font: font, .foregroundColor: Theme.textSecondary]))
        return attr
    }

    private func chipWidth(symbol: String, text: String, font: NSFont) -> CGFloat {
        fittedWidth(singleLineLabel(footerChipString(symbol: symbol, text: text, font: font))) - labelInset * 2
    }

    private func addFooterChip(symbol: String, text: String, x: CGFloat, y: CGFloat, font: NSFont) -> CGFloat {
        let label = singleLineLabel(footerChipString(symbol: symbol, text: text, font: font))
        let width = fittedWidth(label)
        label.frame = NSRect(x: x - labelInset, y: y, width: width, height: lineHeight(font))
        addSubview(label)
        return width - labelInset * 2
    }

    private func labelWidth(_ text: String, font: NSFont) -> CGFloat {
        fittedWidth(singleLineLabel(text, font: font, color: Theme.textSecondary)) - labelInset * 2
    }

    private func addFooterLabel(_ text: String, x: CGFloat, y: CGFloat, font: NSFont) -> CGFloat {
        let label = singleLineLabel(text, font: font, color: Theme.textSecondary)
        let width = fittedWidth(label)
        label.frame = NSRect(x: x - labelInset, y: y, width: width, height: lineHeight(font))
        addSubview(label)
        return width - labelInset * 2
    }

    @objc private func focusTapped() { onFocus?() }

    // MARK: Interaction

    private func setupTracking() {
        let area = NSTrackingArea(rect: bounds, options: [.mouseEnteredAndExited, .activeAlways, .inVisibleRect], owner: self, userInfo: nil)
        addTrackingArea(area)
        trackingArea = area
    }

    override func mouseEntered(with event: NSEvent) {
        closeButton.alphaValue = 1
        onHoverChanged?(true)
    }

    override func mouseExited(with event: NSEvent) {
        closeButton.alphaValue = 0
        onHoverChanged?(false)
    }

    override func mouseDown(with event: NSEvent) { onFocus?() }

    override func scrollWheel(with event: NSEvent) {
        if event.phase == .began { swipeAccumulated = 0; swipeHandled = false }
        guard !swipeHandled else { return }
        swipeAccumulated += event.scrollingDeltaX
        let threshold: CGFloat = 40
        if swipeAccumulated * exitDirection > threshold {
            swipeHandled = true
            onSwipeDismiss?()
        }
    }

    // MARK: Measurement

    private func lineHeight(_ font: NSFont) -> CGFloat { ceil(font.ascender - font.descender + font.leading) }

    private func currentTimeString() -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: model.card.createdAt)
    }
}
