#!/usr/bin/env bash
# Source-able env for the Linux Wails regression bed (docs/wails-linux-smoke.md).
# Usage: source scripts/linux-smoke-env.sh
# Do not use set -e here — this file is meant to be sourced into an interactive shell.
export PATH="/home/box/sdk/node-v22.19.0-linux-x64/bin:/home/box/bin:/home/box/sdk/go1.27.0/bin:/home/box/go/bin:${PATH:-/usr/bin:/bin}"
export GOPATH="/home/box/go"
export DISPLAY="${DISPLAY:-:8}"
export GTK_A11Y="${GTK_A11Y:-none}"
hash -r 2>/dev/null || true

if [ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
  eval "$(dbus-launch --sh-syntax)"
fi

if ! pgrep -x dunst >/dev/null 2>&1; then
  dunst >/tmp/dunst-smoke.log 2>&1 &
fi

if command -v dbus-send >/dev/null 2>&1; then
  if ! dbus-send --session --dest=org.freedesktop.DBus --type=method_call --print-reply \
      /org/freedesktop/DBus org.freedesktop.DBus.NameHasOwner \
      string:org.kde.StatusNotifierWatcher 2>/dev/null | grep -q "boolean true"; then
    if command -v gtk-sni-tray-standalone >/dev/null 2>&1; then
      nohup gtk-sni-tray-standalone -w --bottom --no-strut -s 24 >/tmp/gtk-sni-tray.log 2>&1 &
    else
      echo "linux-smoke-env: gtk-sni-tray-standalone missing; apt install haskell-gtk-sni-tray-utils" >&2
    fi
  fi
fi

_login_node="$(bash -lc 'node -v 2>/dev/null' || true)"
if ! printf '%s' "$_login_node" | grep -Eq '^v(22\.|2[3-9]\.|[3-9])'; then
  echo "linux-smoke-env: WARNING bash -lc node is '${_login_node:-missing}' (need 22+); keep ~/bin/node shim + scripts/zz-dsh-node22.sh.example in profile.d" >&2
fi
echo "linux-smoke-env: node=$(command -v node) $(node -v 2>/dev/null) bash-lc=${_login_node:-missing} display=$DISPLAY"
