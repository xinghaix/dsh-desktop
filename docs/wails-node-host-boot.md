# Node-first Cordis Host boot (no app.whenReady)

Branch: feat/wails3-shell
Goal: Wails HostSidecar prefers a Cordis Host that does not wait on Electron app.whenReady and does not construct BrowserWindow/Tray.

Verified against dsh-plugin-desktop/src/main.ts, electron-runtime.ts, bin.ts, wails-host-sidecar.ts, wails/hostbootstrap.go.

## Current hybrid (H1)

| Step | Needs whenReady / Electron GUI? | Notes |
| --- | --- | --- |
| Import electron in main.ts | Electron process required | Blocks plain node lib/main.js |
| app.requestSingleInstanceLock | Electron app API | No display |
| app.getPath(userData/home/...) | Works before ready | Node: host-paths / bin.defaultDesktopUserDataDirectory |
| crashReporter | Electron-only | Node Host skips |
| ElectronDesktopRuntime ctor | app.getPath | Swap for NodeDesktopRuntime |
| await app.whenReady() (~618) | YES — hard gate today | Sidecar still awaits |
| Shell env + profile prep | No | Pure Node capable |
| process.versions.electron required (~675) | Electron or peer stub | npm_config_target for native deps |
| Profile admission / Setup / Recovery windows | Yes unless sidecar skip | Node mirrors sidecar auto-skip |
| prepareDesktopProfile + materialize | No GUI; Electron ABI target | Native addons still Electron-built |
| safeStorage LAN HTTPS key seal | Electron secret store | Node: protector unavailable |
| boot() + provide desktopRuntime | Needs DesktopRuntime | Node stub OK |
| runtime.schedule via desktop-shell | Registers generation | Node stores spec, never mounts |
| DSH_HOST_READY / auth / LAN announce | No | wails-host-sidecar.ts |
| runtime.mountScheduled | Electron shell | Sidecar + Node skip |

## Electron force-points (honest)

1. app.whenReady in main.ts — sidecar still awaits before Host boot.
2. ElectronDesktopRuntime — BrowserWindow, Tray, dialog, nativeTheme, net.fetch, notifications.
3. safeStorage — LAN HTTPS private-key protection.
4. crashReporter / AppUserModelId / dock — product identity / dumps.
5. process.versions.electron + npm_config_target — materialization targets Electron headers; plain Node can set peer version string but Electron-ABI native addons will not load in stock Node.
6. Pre-Host Electron windows — Setup/Recovery/Profile dialogs (already skipped in Wails sidecar).

## Preferred launch order

1. node lib/host-main.js --dsh-wails-host-sidecar (true Node; no Electron binary).
2. ELECTRON_RUN_AS_NODE=1 <electron> lib/host-main.js --dsh-wails-host-sidecar (Electron ABI, no Chromium/whenReady).
3. Fallback: node lib/bin.js --dsh-wails-host-sidecar → Electron main.js (still whenReady, Electron-light GUI).

Wails defaultHostBootstrap() prefers (1) when lib/host-main.js exists.

## DesktopRuntime: Host vs shell

| API | Host needed? | Node stub |
| --- | --- | --- |
| platform/locale/windowsBuild | Yes | process / LANG |
| updates (request/notify/versions) | plugins | fetch; notify→stderr; isPackaged=false |
| schedule/disposer | desktop-shell | store spec |
| mountScheduled/show/devtools | shell only | no-op / reject |
| registerTrayItem | tray plugins | no-op registration |
| openTerminal/pickDirectory/profile window | actions | log + no-op / null |
| requestRestart/prepareToQuit | settings/shutdown | process exit / relaunch hook |

## Out of scope (Node Host v1)

- Full Recovery/Setup UI (Wails Aux owns hybrid UX).
- Sealed LAN HTTPS CA keys without Wails secret store.
- Replacing electron-builder installers.
