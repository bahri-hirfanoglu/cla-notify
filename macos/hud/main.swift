import AppKit

let version = "1.0.0"

func usage() -> Never {
    let text = """
    cla-notify-hud \(version), macOS card renderer for cla-notify

    usage:
      cla-notify-hud show <card.json>
                                draw the card, stack it, time it out and play its sound
      cla-notify-hud focus <card.json>
                                bring the card's terminal to the front without drawing
      cla-notify-hud render <card.json> <out.png> [dark|light] [width]
                                draw a card offscreen to a PNG for design review
      cla-notify-hud version

    """
    FileHandle.standardError.write(Data(text.utf8))
    exit(64)
}

let args = Array(CommandLine.arguments.dropFirst())

switch args.first {
case "show":
    guard args.count >= 2 else { usage() }
    HUD.run(cardPath: args[1])
case "focus":
    guard args.count >= 2 else { usage() }
    HUD.focus(cardPath: args[1])
case "render":
    guard args.count >= 3 else { usage() }
    HUD.render(cardPath: args[1], outPath: args[2], theme: args.count >= 4 ? args[3] : "dark",
               width: args.count >= 5 ? CGFloat(Double(args[4]) ?? 380) : 380)
case "version", "--version", "-v":
    print(version)
default:
    usage()
}
