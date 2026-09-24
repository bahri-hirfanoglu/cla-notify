#!/usr/bin/env bash
# Container e2e for the linux presenter, driven against a real dbus-daemon --session.
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "== e2e-linux: building and running in golang:1.25 =="

docker run --rm -v "$ROOT_DIR":/src -w /src golang:1.25 bash -c '
set -eu

echo "-- installing dbus --"
apt-get update -qq
apt-get install -y -qq dbus

echo "-- building the e2e binaries --"
go build -buildvcs=false -o /tmp/cla-notify ./cmd/cla-notify
go build -buildvcs=false -o /tmp/fakenotify ./scripts/e2e-linux/fakenotify
go build -buildvcs=false -o /tmp/check ./scripts/e2e-linux/check

echo "-- starting a private session bus --"
DBUS_SESSION_BUS_ADDRESS="$(dbus-daemon --session --fork --print-address)"
export DBUS_SESSION_BUS_ADDRESS

export CLA_NOTIFY_STATE_DIR="$(mktemp -d)"
export FAKE_NOTIFY_LOG="$(mktemp)"

/tmp/fakenotify &
FAKENOTIFY_PID=$!
trap "kill $FAKENOTIFY_PID 2>/dev/null || true" EXIT

echo "-- waiting for the fake notification service to own its bus name --"
for i in $(seq 1 50); do
  if dbus-send --session --print-reply --dest=org.freedesktop.Notifications \
      /org/freedesktop/Notifications org.freedesktop.Notifications.GetServerInformation \
      >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

FAKE_BIN="$(mktemp -d)"
cp testdata/linux/bin/tmux "$FAKE_BIN/tmux"
chmod +x "$FAKE_BIN/tmux"
export TMUX_CALL_LOG="$(mktemp)"
export PATH="$FAKE_BIN:$PATH"

# RunLinuxWorker deletes the card file when done, so drive it against copies, never the fixtures.
SCRATCH="$(mktemp -d)"
cp testdata/linux/cards/*.json "$SCRATCH/"

echo "-- replace_id scenario: two cards for the same session --"
/tmp/cla-notify present-linux "$SCRATCH/replace-1.json" &
PID1=$!
sleep 0.5
/tmp/cla-notify present-linux "$SCRATCH/replace-2.json"
wait "$PID1"

echo "-- maxCards scenario: four overlapping cards, maxCards is 3 --"
/tmp/cla-notify present-linux "$SCRATCH/max-1.json" &
PID1=$!
sleep 0.15
/tmp/cla-notify present-linux "$SCRATCH/max-2.json" &
PID2=$!
sleep 0.15
/tmp/cla-notify present-linux "$SCRATCH/max-3.json" &
PID3=$!
sleep 0.15
/tmp/cla-notify present-linux "$SCRATCH/max-4.json"
wait "$PID1" "$PID2" "$PID3"

echo "-- click scenario: ActionInvoked(focus) should raise the terminal --"
/tmp/cla-notify present-linux "$SCRATCH/click.json"

echo "-- hook scenario: cla-notify hook drives a real hook event end to end --"
HOOK_STATE_DIR="$(mktemp -d)"
HOOK_CONFIG_DIR="$(mktemp -d)"
echo "{\"labels\":{\"focusButton\":\"Back to Terminal\"}}" > "$HOOK_CONFIG_DIR/config.json"
CLA_NOTIFY_STATE_DIR="$HOOK_STATE_DIR" CLA_NOTIFY_CONFIG="$HOOK_CONFIG_DIR/config.json" \
  /tmp/cla-notify hook < internal/hook/testdata/permission.json
sleep 1

echo "-- assertions --"
/tmp/check
'
STATUS=$?

if [ "$STATUS" -eq 0 ]; then
  echo "e2e-linux.sh: PASS"
else
  echo "e2e-linux.sh: FAIL (exit $STATUS)"
fi
exit "$STATUS"
