#!/bin/sh
# Cross-compiles every Go binary into dist/, using docker's golang:1.25 image unless
# a local `go` is on PATH. Then builds the macOS Swift HUD when running on darwin.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
version=$(cat "$root/VERSION")
ldflags="-s -w -X github.com/bahri-hirfanoglu/cla-notify/internal/cli.Version=$version"

mkdir -p "$root/dist"

go_build() {
  goos=$1
  goarch=$2
  out=$3
  if command -v go >/dev/null 2>&1; then
    (cd "$root" && GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
      go build -trimpath -ldflags "$ldflags" -o "$out" ./cmd/cla-notify)
  else
    docker run --rm -v "$root":/src -w /src \
      -e GOOS="$goos" -e GOARCH="$goarch" -e CGO_ENABLED=0 \
      golang:1.25 go build -trimpath -ldflags "$ldflags" -o "/src/dist/$(basename "$out")" ./cmd/cla-notify
  fi
}

echo "-- building darwin/amd64 and darwin/arm64 --"
go_build darwin amd64 "$root/dist/cla-notify-darwin-amd64"
go_build darwin arm64 "$root/dist/cla-notify-darwin-arm64"

if command -v lipo >/dev/null 2>&1; then
  lipo -create "$root/dist/cla-notify-darwin-amd64" "$root/dist/cla-notify-darwin-arm64" \
    -output "$root/dist/cla-notify-darwin"
  rm -f "$root/dist/cla-notify-darwin-amd64" "$root/dist/cla-notify-darwin-arm64"
else
  echo "no lipo on this host, leaving separate darwin-amd64/darwin-arm64 binaries" >&2
fi

echo "-- building linux/amd64 and linux/arm64 --"
go_build linux amd64 "$root/dist/cla-notify-linux-amd64"
go_build linux arm64 "$root/dist/cla-notify-linux-arm64"

echo "-- building windows/amd64 and windows/arm64 --"
go_build windows amd64 "$root/dist/cla-notify-windows-amd64.exe"
go_build windows arm64 "$root/dist/cla-notify-windows-arm64.exe"

if [ "$(uname -s)" = "Darwin" ]; then
  echo "-- building the macOS HUD --"
  "$root/scripts/build-hud.sh"
fi

echo "-- done --"
ls -la "$root/dist"
