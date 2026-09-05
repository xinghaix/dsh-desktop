# Electron shell fallback (LAST-RESORT)

Primary: Wails + Node Host (start:wails / start:host).
LAST-RESORT: DSH_HOST_LAUNCHER=electron-main with DSH_ALLOW_ELECTRON_MAIN=1.
Prefer electron-as-node when ABI natives require Electron runtime without BrowserWindow.

## Triggers
- node: default when lib/host-main.js exists
- electron-as-node: profile node_modules Electron ABI or DSH_HOST_LAUNCHER=electron-as-node
- electron-main: LAST-RESORT only with explicit launcher and/or DSH_ALLOW_ELECTRON_MAIN
- missing host-main.js may fall back toward electron-main; rebuild instead

## Pressure-test
1. DSH_HOST_LAUNCHER=node start:host
2. electron-as-node with ABI natives
3. LAST-RESORT electron-main + allow flag (document GUI limits on Linux bed)
4. electron-builder product start still works

Do not mass-delete src/main.ts; product CI needs it until release flip.
See docs/wails-migration.md and src/wails-shell-bridge.md.
