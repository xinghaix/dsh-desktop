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
- `dsh --profile web` is the Host/Web UI Cordis stack; Wails loads its UI from a **user install**. Packaged `host-main` is not the product default (`DSH_ALLOW_PACKAGED_HOST=1` for dev only).
- Design note: docs/evidence/wails-user-installed-dsh-20260905.md




## What “Host” means

Host is the Cordis process that serves the Web UI — the same stack users get from `dsh --profile web` (or their existing profile). Wails is a native shell that loads that Host UI URL from a **user-installed** dsh. Desktop control features should be tools/plugins on that install, not a bundled Host.

## Architecture split: home CLI vs packaged Host launch

- **Wails Host launch (desktop product):** user-installed only — `DSH_BIN` → `DSH_HOME` / chosen dir → home roots → PATH. Packaged/monorepo `host-main` only with `DSH_ALLOW_PACKAGED_HOST=1` (dev). Missing → install-help page. Profile: existing user profile or `web`.
- **`dsh` CLI (`desktop-cli` / terminal shims):** home-first — `DSH_CLI_BIN` → `DSH_HOME` / `~/.dsh`|`~/dsh`|XDG `…/@deepseek-ai/dsh/lib/bin.js`. No silent ASAR fallback. Opt-in only: `DSH_CLI_ALLOW_BUNDLED=1`.
- Do not use the desktop-bundled CLI/runtime to impersonate the user’s home `dsh`.
- Evidence: `docs/evidence/wails-cli-home-vs-host-packaged-20260905.md`
