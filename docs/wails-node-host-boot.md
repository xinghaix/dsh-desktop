# Node-first Cordis Host boot

## Preferred launch order

1. Stock Node host-main
2. Electron-as-Node when Electron ABI natives are present
3. LAST RESORT Electron main (DSH_HOST_LAUNCHER=electron-main; may need DSH_ALLOW_ELECTRON_MAIN=1)

## Env

- DSH_HOST_LAUNCHER
- DSH_HOST_ELECTRON_AS_NODE
- DSH_ALLOW_ELECTRON_MAIN
- ELECTRON_PATH

See host-launcher.ts / bin.ts / hostbootstrap.go.

## Crash evidence

Node Host: active-run.json + uncaughtException dumps under crash-evidence/ (no Crashpad).
Wails shell: same directory family via crash_evidence.go.

## Identity / dock

- Windows AppUserModelID via CapabilitiesService
- macOS: Flash + tray tooltip; numeric Dock badge blocked in Wails v3 beta
