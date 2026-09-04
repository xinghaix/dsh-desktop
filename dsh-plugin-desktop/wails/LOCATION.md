# DSH Desktop — Wails v3 native shell

This directory owns the **Go + Wails v3** desktop shell that will replace Electron's `BrowserWindow` / Tray / Menu / Dialog surface.

Location rationale: lives under `dsh-plugin-desktop/` beside the existing Cordis Host / Electron adapter (`src/`), so the Node host package and the native shell stay one product package while the shell can evolve independently.

See `README.md` in this folder for build/run. Migration notes land in `docs/wails-migration.md` (added in a later commit).

