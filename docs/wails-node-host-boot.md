# Node-first Cordis Host boot

Branch: feat/wails3-shell

## Preferred launch order

1. Stock Node host-main
2. Electron-as-Node when needed
3. LAST RESORT Electron main

## Env
- DSH_HOST_LAUNCHER
- DSH_HOST_ELECTRON_AS_NODE
- ELECTRON_PATH
See host-launcher.ts / bin.ts / hostbootstrap.go.
Scripts: start:host, start:host:node, start:host:electron-as-node.

## macOS
- start:host:node (fresh)
- start:host:electron-as-node (after materialization)
- start:wails (hybrid)

## LAN HTTPS / identity
- Node AES-GCM protector (wails/LAN-HTTPS.md)
- Windows AppUserModelID via CapabilitiesService
- crashReporter blocked
## Fallback triggers
1. Missing host-main.js
2. Forced main launcher mode
