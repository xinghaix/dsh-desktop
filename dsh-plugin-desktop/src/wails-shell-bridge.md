# ElectronDesktopRuntime → Wails shell bridge

Hybrid migration map (do not edit `deepseek-harness/`).

| Electron surface (`ElectronDesktopRuntime` / `ElectronShellGeneration`) | Wails v3 owner | Status |
| --- | --- | --- |
| `BrowserWindow` + `loadURL(spec.url)` | `application.Window` + `SetURL` / `ShellService.LoadHostURL` | Implemented in `wails/` |
| `Tray` + context menu | `app.SystemTray.New` + `SetMenu` | Implemented |
| Application / context menus | `app.NewMenu` / `app.Menu.Set` | Implemented (subset) |
| `dialog.showOpenDialog` / message boxes | `app.Dialog.OpenFile` / `Info` / `Question` | Implemented (subset) |
| Cordis `boot()` in `main.ts` | Auto-started Host sidecar (`HostSidecar` default → desktop start + `--dsh-wails-host-sidecar`) | Hybrid — Host still Electron/Node; Wails loads announced URL |
| Preload / `session.fetch` auth cookie | BridgeService + announceWailsHostAuthHeader; sidecar enables ordinary loopback browser access; WebKitGTK cannot inject per-request renderer header | Partial (hybrid) |
| Profile / setup / recovery auxiliary windows | Wails AuxWindowService + frontend/dist/aux HTML; full React native-ui still Electron | Partial (hybrid) |
| Updates, notifications, terminal, workspace admission | CapabilitiesService covers notify/export/terminal/update-probe; full update install + packaged terminal + LAN HTTPS still Electron | Partial (hybrid) |
| Packaging (`electron-builder`) | build:wails/start:wails/package:wails wired; electron-builder remains CI/fallback default | Partial |

Preferred hybrid loop today:

1. Run the Wails shell (`go run .`). It auto-starts the existing desktop entry with `--dsh-wails-host-sidecar` unless `-no-host` / `DSH_HOST_AUTOSTART=0`.
2. Electron `main.ts` boots Cordis as usual, then announces `DSH_HOST_READY <authenticated-loopback-url>` instead of mounting BrowserWindow/Tray.
3. Wails navigates to the announced URL (`ShellService.LoadHostURL`). Manual overrides: `-host-url`, `DSH_HOST_URL`, `DSH_HOST_COMMAND`, `announce-host-ready.mjs`.
