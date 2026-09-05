# DSH Desktop — Wails v3 native shell

This directory owns the **primary Go + Wails v3** desktop shell (window / tray / menu / dialog). Electron BrowserWindow / Tray remains last-resort fallback only.

Location rationale: lives under `dsh-plugin-desktop/` beside the existing Cordis Host / Electron adapter (`src/`), so the Node host package and the native shell stay one product package while the shell can evolve independently.

See `README.md` in this folder for build/run. Migration notes: `../../../docs/wails-migration.md`, `../src/wails-shell-bridge.md`, `../../../docs/wails-node-host-boot.md`, `../docs/electron-shell-fallback.md`.

