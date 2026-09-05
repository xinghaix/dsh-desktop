# Electron to Wails v3 migration

Branch: `feat/wails3-shell`
Shell path: `dsh-plugin-desktop/wails/`
Go: 1.27 · Wails: v3.0.0-beta.16

## Strategy

Prefer a working hybrid over a fake full rewrite:

1. Wails owns the native window, tray, menus, and dialogs.
2. Node/Electron Cordis Host (today's `dsh-plugin-desktop/src/main.ts` boot path) continues to materialize profiles, serve the web UI on loopback, and mint the authenticated browser URL (`desktopLoopbackBrowserUrl` + `connection.authenticatedUrl`).
3. Wails Host-sidecar mode (`--dsh-wails-host-sidecar` / `DSH_WAILS_HOST_SIDECAR=1`) skips Electron BrowserWindow/Tray and announces the UI URL.
4. Wails navigates to that URL (`ShellService.LoadHostURL` / `HostSidecar` auto-start), preferring a loopback **AuthProxy** that injects `x-dsh-desktop-renderer`.

`deepseek-harness/` remains an unmodified submodule.

## Build / run

See `dsh-plugin-desktop/wails/README.md` and `docs/wails-workspace-scripts.md`.

## Surface map

Canonical table: `dsh-plugin-desktop/src/wails-shell-bridge.md`.

## Progress (Stages A–E + hard debt H1–H6)

### Done in Wails / hybrid

- Main webview window with remote URL load (`-host-url` / `DSH_HOST_URL`)
- Hide-to-tray + system tray menu; application menu subset
- Native directory picker + info dialogs; generated bindings + control UI
- Host sidecar discovery + default auto-start of existing desktop start path
- Electron Host-sidecar announce (`DSH_HOST_READY`, auth header, LAN HTTPS, recovery required)
- Auxiliary windows: setup / profile / recovery (hybrid shell-owned HTML under `frontend/dist/aux/`)
- Auth/IPC `BridgeService` + `bridge.js`; **AuthProxy** loopback header injection (H2) — see `wails/AUTH.md`
- **H1 Electron-light Host**: sidecar skips Setup/Recovery/profile-compat BrowserWindows; preserves sidecar argv on relaunch; forwards markers via `bin.ts`
- CapabilitiesService: notifications, save/export, reveal-in-folder, system terminal fallback
- **H3** update check against public version endpoint + macOS/Windows download/open installers
- **H4** LAN HTTPS status bridge from Host announce (TLS still Host-terminated) — see `wails/LAN-HTTPS.md`
- **H5** `run-wails.mjs` portable Go (`DSH_GO_BIN` → PATH → SDK layouts); `package:wails` fallback; `smoke:wails`; `docs/wails-ci-smoke.yml.example` (install when credential has workflow scope)
- Packaging scripts: `build:wails`, `start:wails`, `dev:wails`, `package:wails`, `smoke:wails`

### Still Electron / Host-owned (blocked or deferred)

- Full Cordis bootstrap still requires Electron `main.ts` (`app.whenReady`, `ElectronDesktopRuntime`). Plain Node Host entry is not viable yet without rewriting `DesktopRuntime`.
- Native webview per-request header hooks remain unavailable in Wails v3 beta on Mac/Linux/Windows public APIs (AuthProxy is the workaround).
- Full React native-ui Recovery/Setup/Profile documents and checkpoint uninstall UX
- Packaged DSH terminal shims; Linux update installers
- Certificate private-key protection via Electron `safeStorage` for LAN HTTPS
- macOS dock identity, Windows AppUserModelId, crashReporter
- Production electron-builder installers remain CI default; `wails3 package` AppImage needs `file`/`appimagetool` deps on the host (fallback go binary works)
- Headless product CI (`ci.yml`) remains Electron/Yarn based; Wails smoke is additive only

### Blocked on this headless Linux box

- No interactive GUI verification of tray/notifications/aux windows
- End-to-end AppImage/`wails3 package` may fail without `file` package / full linuxdeploy deps
- Cannot fully exercise Electron Host boot without a display (sidecar still needs `app.whenReady`)

## Next slices (suggested)

1. Extract a true Node-only Cordis Host entry (replace `ElectronDesktopRuntime` adapters).
2. Upstream Wails public request-header hooks; then retire ordinary-browser fallback where AuthProxy is enough.
3. Wire Wails-native toggles for LAN HTTPS enable + CA install UX.
4. Make `wails3 package` the release default per platform; keep electron-builder as rollback.
