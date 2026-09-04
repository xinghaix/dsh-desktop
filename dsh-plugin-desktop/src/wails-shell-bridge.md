# ElectronDesktopRuntime → Wails shell bridge

Hybrid migration map (do not edit `deepseek-harness/`).

| Electron surface (`ElectronDesktopRuntime` / `ElectronShellGeneration`) | Wails v3 owner | Status |
| --- | --- | --- |
| `BrowserWindow` + `loadURL(spec.url)` | `application.Window` + `SetURL` / `ShellService.LoadHostURL` | Implemented in `wails/` |
| `Tray` + context menu | `app.SystemTray.New` + `SetMenu` | Implemented |
| Application / context menus | `app.NewMenu` / `app.Menu.Set` | Implemented (subset) |
| `dialog.showOpenDialog` / message boxes | `app.Dialog.OpenFile` / `Info` / `Question` | Implemented (subset) |
| Cordis `boot()` in `main.ts` | Node sidecar via `HostSidecar` (`DSH_HOST_COMMAND` / `DSH_HOST_URL` / `DSH_HOST_URL_FILE`) | Hybrid — Host still Node |
| Preload / `session.fetch` auth cookie | Still Electron Host path; Wails loads authenticated URL when Host announces it | Debt |
| Profile / setup / recovery auxiliary windows | Still Electron `BrowserWindow` HTML UIs | Debt |
| Updates, notifications, terminal, workspace admission | Still Electron adapters | Debt |
| Packaging (`electron-builder`) | Future `wails3 package` | Debt |

Preferred hybrid loop today:

1. Start Cordis Host the existing way (or a future headless entry) until the loopback web server is up.
2. Announce `http://127.0.0.1:<port>/` with `wails/scripts/announce-host-ready.mjs` (or set `DSH_HOST_URL`).
3. Run the Wails shell (`go run .` / `wails3 build`) which navigates the native webview to that URL.
