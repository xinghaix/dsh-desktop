# ElectronDesktopRuntime → Wails shell bridge

Hybrid migration map (do not edit `deepseek-harness/`).

| Electron surface (`ElectronDesktopRuntime` / `ElectronShellGeneration`) | Wails v3 owner | Status |
| --- | --- | --- |
| `BrowserWindow` + `loadURL(spec.url)` | `application.Window` + `SetURL` / `ShellService.LoadHostURL` | Implemented in `wails/` |
| `Tray` + context menu | `app.SystemTray.New` + `SetMenu` | Implemented |
| Application / context menus | `app.NewMenu` / `app.Menu.Set` | Implemented (subset) |
| `dialog.showOpenDialog` / message boxes | `app.Dialog.OpenFile` / `Info` / `Question` | Implemented (subset) |
| Cordis `boot()` in `main.ts` | Auto-started Host sidecar (`HostSidecar` default → desktop start + `--dsh-wails-host-sidecar`) | Hybrid — Host still Electron/Node; Wails loads announced URL |
| Preload / `session.fetch` auth cookie | Sidecar announces authenticated loopback URL; renderer header injection still missing | Partial |
| Profile / setup / recovery auxiliary windows | Still Electron `BrowserWindow` HTML UIs | Debt |
| Updates, notifications, terminal, workspace admission | Still Electron adapters | Debt |
| Packaging (`electron-builder`) | Future `wails3 package` | Debt |

Preferred hybrid loop today:

1. Run the Wails shell (`go run .`). It auto-starts the existing desktop entry with `--dsh-wails-host-sidecar` unless `-no-host` / `DSH_HOST_AUTOSTART=0`.
2. Electron `main.ts` boots Cordis as usual, then announces `DSH_HOST_READY <authenticated-loopback-url>` instead of mounting BrowserWindow/Tray.
3. Wails navigates to the announced URL (`ShellService.LoadHostURL`). Manual overrides: `-host-url`, `DSH_HOST_URL`, `DSH_HOST_COMMAND`, `announce-host-ready.mjs`.
