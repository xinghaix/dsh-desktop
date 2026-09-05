#!/usr/bin/env bash
# Install Node 22 ~/bin shims for the Linux Wails smoke bed. Safe to re-run.
# Copy zz-dsh-node22.sh.example to /etc/profile.d/zz-dsh-node22.sh as root when needed.
set -euo pipefail
NODE_SDK="${DSH_NODE_SDK-/home/box/sdk/node-v22.19.0-linux-x64/bin}"
BIN_DIR="${DSH_NODE_BIN-${HOME}/bin}"
mkdir -p "$BIN_DIR"
for b in node npm npx corepack; do
  if [ -x "$NODE_SDK/$b" ]; then
    ln -sfn "$NODE_SDK/$b" "$BIN_DIR/$b"
  fi
done
echo "install-linux-smoke-node22: shims in $BIN_DIR"
bash -lc 'node -v' || true
