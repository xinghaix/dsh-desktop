# ElectronDesktopRuntime → Wails shell bridge

Hybrid migration map (do not edit `deepseek-harness/`)
.

| Electron surface | Wails v3 owner | Status |
| --- | --- | --- |
| BrowserWindow + loadURL | application.Window + ShellService.LoadHostURL | Implemented |
| Tray + context menu | SystemTray | Implemented |
| Application menus | app.NewMenu | Implemented (subset) |
| Native dialogs | app.Dialog | Implemented (subset) |
| Cordis boot | host-launcher auto (Node / Electron-as-Node / LAST-RESORT main) | Node Host no whenReady; Electron-as-Node for ABI natives; main still whenReady |
| Preload / session.fetch auth | BridgeService + AuthProxy (H2) | AuthProxy default production; native hooks blocked |
| Setup/profile/recovery windows | AuxWindowService + native-ui prefer + scheme bridge | Hybrid |
| Updates / notifications / terminal | CapabilitiesService check+download (H3) | Implemented (mac/win/linux AppImage path) |
| LAN HTTPS edge | Host TLS + DSH_HOST_LAN_HTTPS announce (H4) | Partial |
| Packaging | package/smoke:wails + wails-smoke.yml (H5) | Partial; electron-builder still CI default |

Preferred hybrid loop:

1. start:wails / hostbootstrap auto-select Node or Electron-as-Node host-main.
2. Host announces DSH_HOST_READY + auth header (+ LAN HTTPS).
3. Wails prefers AuthProxy URL then LoadHostURL.
