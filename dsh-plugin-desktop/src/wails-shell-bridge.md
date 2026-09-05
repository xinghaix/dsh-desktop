# ElectronDesktopRuntime → Wails shell bridge

Hybrid migration map (do not edit `deepseek-harness/`)
.

| Electron surface | Wails v3 owner | Status |
| --- | --- | --- |
| BrowserWindow + loadURL | application.Window + ShellService.LoadHostURL | Implemented |
| Tray + context menu | SystemTray | Implemented |
| Application menus | app.NewMenu | Implemented (subset) |
| Native dialogs | app.Dialog | Implemented (subset) |
| Cordis boot in main.ts | HostSidecar Electron-light (H1) | Hybrid; still needs app.whenReady |
| Preload / session.fetch auth | BridgeService + AuthProxy (H2) | Partial; native hooks unavailable |
| Setup/profile/recovery windows | AuxWindowService; Electron skipped in sidecar | Partial |
| Updates / notifications / terminal | CapabilitiesService check+download (H3) | Partial; Linux install deferred |
| LAN HTTPS edge | Host TLS + DSH_HOST_LAN_HTTPS announce (H4) | Partial |
| Packaging | package/smoke:wails + wails-smoke.yml (H5) | Partial; electron-builder still CI default |

Preferred hybrid loop:

1. start:wails auto-starts Host with --dsh-wails-host-sidecar.
2. Electron-light Host announces DSH_HOST_READY + auth header (+ LAN HTTPS).
3. Wails prefers AuthProxy URL then LoadHostURL.
