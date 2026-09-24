#!/bin/sh
# Builds dist/cla-notify-hud as a universal (arm64 + x86_64) binary targeting macOS 12+.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
out="$root/dist/cla-notify-hud"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir -p "$root/dist"

for arch in arm64 x86_64; do
  swiftc -O -whole-module-optimization \
    -target "$arch-apple-macos12.0" \
    -framework AppKit \
    -o "$work/cla-notify-hud-$arch" \
    "$root"/macos/hud/*.swift
done

lipo -create "$work/cla-notify-hud-arm64" "$work/cla-notify-hud-x86_64" -output "$out"
codesign --force --sign - "$out"
chmod +x "$out"
lipo -info "$out"
