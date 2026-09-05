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

## User-installed / AppImage Host

- DSH_BIN: Desktop Host JS entry or dsh-desktop executable (Wails sidecar mode).
- PATH: dsh-desktop or dsh-plugin-desktop when monorepo layout is missing.
- DSH_HOST_URL / DSH_HOST_COMMAND: attach or spawn an already chosen Host.
- Bare public dsh CLI is not a Desktop Host.
- Design note: docs/evidence/wails-user-installed-dsh-20260905.md


## User dsh home-first (feat/wails-user-dsh-home)

Order: DSH_BIN -> DSH_HOME -> ~/.dsh|~/dsh|XDG -> ~/.local/bin -> PATH -> monorepo fallback.
Friendly missing copy lists checked paths. See docs/evidence/wails-user-dsh-home-20260905.md.
