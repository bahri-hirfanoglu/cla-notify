#!/usr/bin/env bash
# Table test for bin/cla-notify's OS/arch detection: stubs uname and dist/ so no real binaries are needed.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/bin" "$work/dist" "$work/fakebin"
cp "$root/bin/cla-notify" "$work/bin/cla-notify"
chmod +x "$work/bin/cla-notify"

cat > "$work/fakebin/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) printf '%s\n' "$UNAME_S" ;;
  -m) printf '%s\n' "$UNAME_M" ;;
  *) exit 1 ;;
esac
EOF
chmod +x "$work/fakebin/uname"

# Every dist stub just prints its own basename, so the launcher's chosen binary is visible on stdout.
for name in cla-notify-darwin cla-notify-linux-amd64 cla-notify-linux-arm64 \
  cla-notify-windows-amd64.exe cla-notify-windows-arm64.exe; do
  path="$work/dist/$name"
  {
    echo '#!/bin/sh'
    echo "echo $name"
  } > "$path"
  chmod +x "$path"
done

export PATH="$work/fakebin:$PATH"

failures=0

# name, uname -s, uname -m, expected dist basename ("" means no binary for this platform)
cases='
macos-arm64|Darwin|arm64|cla-notify-darwin
macos-intel|Darwin|x86_64|cla-notify-darwin
linux-x64|Linux|x86_64|cla-notify-linux-amd64
linux-arm64-aarch64|Linux|aarch64|cla-notify-linux-arm64
linux-arm64-arm64|Linux|arm64|cla-notify-linux-arm64
windows-x64-gitbash|MINGW64_NT-10.0-19045|x86_64|cla-notify-windows-amd64.exe
windows-arm64-gitbash-aarch64|MINGW64_NT-10.0-22631|aarch64|cla-notify-windows-arm64.exe
windows-arm64-gitbash-uppercase|MINGW64_NT-10.0-22631|ARM64|cla-notify-windows-arm64.exe
windows-arm64-emulated-x64|MINGW64_NT-10.0-22631|x86_64|cla-notify-windows-amd64.exe
unknown-os|SunOS|x86_64|
'

while IFS='|' read -r name uname_s uname_m expect; do
  [ -n "$name" ] || continue
  export UNAME_S="$uname_s" UNAME_M="$uname_m"

  if [ -n "$expect" ]; then
    got=$("$work/bin/cla-notify" version)
    if [ "$got" != "$expect" ]; then
      echo "FAIL $name: got binary $got, want $expect"
      failures=$((failures + 1))
    else
      echo "ok $name -> $expect"
    fi
  else
    if err=$("$work/bin/cla-notify" version 2>&1); then
      echo "FAIL $name: expected failure for an unsupported platform, got: $err"
      failures=$((failures + 1))
    else
      echo "ok $name -> no binary, exits nonzero"
    fi
    # A hook invocation must stay silent and exit 0 even with no binary for this platform.
    if ! out=$(printf '{}' | "$work/bin/cla-notify" hook 2>/tmp/test-launcher-hook-err.$$); then
      echo "FAIL $name: hook subcommand should exit 0 when no binary is available"
      failures=$((failures + 1))
    elif [ -n "$out" ]; then
      echo "FAIL $name: hook subcommand printed to stdout: $out"
      failures=$((failures + 1))
    else
      echo "ok $name -> hook subcommand exits 0 with no stdout"
    fi
    rm -f /tmp/test-launcher-hook-err.$$
  fi
done <<< "$cases"

if [ "$failures" -ne 0 ]; then
  echo "test-launcher.sh: $failures failure(s)"
  exit 1
fi
echo "test-launcher.sh: PASS"
