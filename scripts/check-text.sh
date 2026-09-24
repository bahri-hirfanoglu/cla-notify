#!/usr/bin/env bash
# Fails on U+2014 (em dash) or U+2013 (en dash) in any tracked or untracked, non-ignored text file.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

em=$(printf '\342\200\224')
en=$(printf '\342\200\223')

found=0

git ls-files -z --cached --others --exclude-standard -- . ':!dist/' ':!go.sum' |
while IFS= read -r -d '' f; do
  [ -f "$f" ] || continue
  case "$(file -b --mime-encoding "$f" 2>/dev/null || echo binary)" in
    binary) continue ;;
  esac
  grep -n -F -e "$em" -e "$en" "$f" | sed "s|^|$f:|" || true
done > "$root/.check-text.tmp"

if [ -s "$root/.check-text.tmp" ]; then
  cat "$root/.check-text.tmp"
  found=1
fi
rm -f "$root/.check-text.tmp"

if [ "$found" -ne 0 ]; then
  echo "found em dash (U+2014) or en dash (U+2013), replace with a hyphen, comma or new sentence" >&2
  exit 1
fi
exit 0
