# Electron to Wails v3 migration

Branch: `feat/wails3-shell`
Shell path: `dsh-plugin-desktop/wails/`
Go: 1.27 · Wails: v3.0.0-beta.16

## Strategy

Prefer a working hybrid over a fake full rewrite:

1. Wails owns the native window, tray, menus, and dialogs.
2. Node/Electron Cordis Host (today's `dsh-plugin-desktop/src/main.ts` boot path) continues to materialize profiles, serve the web UI on loopback, and mint the authenticated browser URL (`desktopLoopbackBrowserUrl` + `connection.authenticatedUrl`).
3. Wails Host-sidecar mode (`--dsh-wails-host-sidecar` / `DSH_WAILS_HOST_SIDECAR=1`) skips Electron BrowserWindow/Tray and announces the UI URL.
4. Wails navigates to that URL (`ShellService.LoadHostURL` / `HostSidecar` auto-start).

`deepseek-harness/` remains an unmodified submodule.

## Build / run

See `dsh-plugin-desktop/wails/README.md`.

## Surface map

Canonical table: `dsh-plugin-desktop/src/wails-shell-bridge.md`.

### Done in Wails

- Main webview window with remote URL load (`-host-url` / `DSH_HOST_URL`)
- Hide-to-tray + system tray menu
- Application menu (File / View / Help subset)
- Native directory picker + info dialogs
- Generated bindings + control UI
- Host sidecar discovery + default auto-start of existing desktop start path
- Electron Host-sidecar announce (announceWailsHostReady / DSH_HOST_READY)

### Remaining Electron Host-boot debt

- Full Cordis bootstrap still requires Electron `main.ts` (`app.whenReady`, `ElectronDesktopRuntime`, profile/setup/recovery windows).
- Renderer access header / `session.fetch` authentication cookie exchange in `electron-shell-generation.ts`.
- Preload bridge (`preload.ts`) and desktop client IPC.
- Auxiliary BrowserWindows: setup wizard, profile selection/create, recovery, desktop-dialog HTML UIs under `src/native-ui/`.
- Updates (`DesktopUpdateAdapter`), notifications, terminal open, workspace admission / directory picker routing, LAN HTTPS edge.
- macOS dock identity, Windows AppUserModelId, crashReporter, safeStorage.
- Packaging: `electron-builder` scripts under `dsh-plugin-desktop/scripts/` — not yet replaced by `wails3 package`.
- Headless CI remains Electron/Yarn based; Wails graphical smoke is optional.

## Next slices (suggested)

1. Headless Host entry that boots Cordis without constructing `BrowserWindow`.
2. Port renderer access / auth cookie into Wails webview request hooks.
3. Reimplement setup/profile/recovery windows as Wails windows or Host routes.
4. Implement `DesktopRuntime` against Wails services and delete Electron adapters.
5. Switch packaging to `wails3 package` per platform.

### Host auto-start note

Launching the Wails shell without -no-host auto-starts Cordis Host via the existing desktop start path. Overrides include DSH_HOST_URL and DSH_HOST_AUTOSTART=0.
