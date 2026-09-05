# Electron to Wails v3 migration

Branch: feat/wails3-shell
Shell: dsh-plugin-desktop/wails/

## Primary run path

- start:wails / dev:wails / smoke:wails (repo root scripts)
- start:host (Node Host sidecar)
- node scripts/wails-smoke.mjs
- Electron main.ts is LAST-RESORT (docs/electron-shell-fallback.md)

## Done

- Main webview, tray, menus, dialogs
- Host sidecar auto-start (Node / Electron-as-Node / LAST-RESORT Electron main)
- AuthProxy production path (loopback hardened)
- Aux Recovery/Setup/Profile; prefers lib/native-ui + scheme bridge
- Notifications, export, reveal, terminal, updates (mac/win/linux AppImage)
- File-based crash evidence (active-run + panic/exception dumps)
- Dock/taskbar attention via Flash + tray tooltip count
- Windows AppUserModelID; Node AES-GCM LAN HTTPS protector
- Root/workspace wails scripts; scripts/wails-smoke.mjs; docs/wails-ci-smoke.yml.example (workflow push needs workflow scope)

## Permanently platform-blocked

- Native webview per-request header hooks (AuthProxy workaround)
- Electron Crashpad/minidumps (file-based substitute only)
- macOS numeric Dock badge / setBadgeCount (Flash + tooltip only)
- Full Electron Recovery checkpoint uninstall UX without Host controller state
- Interactive GUI verification on headless Linux
- wails3 package AppImage host deps may be missing (go binary fallback OK)
- electron-builder remains default product CI until release flip

Canonical surface map: dsh-plugin-desktop/src/wails-shell-bridge.md
