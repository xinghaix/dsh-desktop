# ElectronDesktopRuntime → Wails shell bridge

Hybrid migration map (do not edit `deepseek-harness/`)
.

| Electron surface | Wails v3 owner | Status |
| --- | --- | --- |
| BrowserWindow + loadURL | application.Window + ShellService.LoadHostURL | Implemented |
| Tray + context menu | SystemTray | Implemented |
| Application menus | app.NewMenu | Implemented (subset) |
| Native dialogs | app.Dialog | Implemented (subset) |
| Cordis boot | Node host-main.ts (preferred) / Electron-light main.ts fallback | Node Host: no whenReady; Electron fallback still needs whenReady |
| Preload / session.fetch auth | BridgeService + AuthProxy (H2) | Partial; native hooks unavailable |
| Setup/profile/recovery windows | AuxWindowService; Electron skipped in sidecar | Partial |
| Updates / notifications / terminal | CapabilitiesService check+download (H3) | Partial; Linux install deferred |
| LAN HTTPS edge | Host TLS + DSH_HOST_LAN_HTTPS announce (H4) | Partial |
| Packaging | package/smoke:wails + wails-smoke.yml (H5) | Partial; electron-builder still CI default |

Preferred hybrid loop:

1. start:wails prefers node lib/host-main.js --dsh-wails-host-sidecar.
2. Node Host (or Electron-light fallback) announces DSH_HOST_READY + auth header (+ LAN HTTPS).
3. Wails prefers AuthProxy URL then LoadHostURL.
