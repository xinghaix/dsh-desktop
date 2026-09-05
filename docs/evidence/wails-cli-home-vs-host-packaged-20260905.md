# CLI home-first; Wails loads Cordis Host UI (2026-09-05)

Branch: `feat/wails-user-dsh-home`

## Concept (aligned with product intent)

- **Host** is the Cordis process that serves the Web UI — the same family as `dsh --profile web` (or the desktop default web-capable profile).
- **Desktop** is primarily a **Wails shell** that loads that Host UI. It may also package/launch a Host process (product path), including desktop plugin composition / sidecar announce when started via `host-main` / `dsh-desktop`.
- **`dsh` CLI** must resolve the **user home** install (`DSH_HOME` / `~/.dsh` / `DSH_CLI_BIN`), not silently the copy bundled inside the desktop package.
- Host and `dsh --profile web` are **not** unrelated products; packaging may add desktop-facing entrypoints (`host-main`, sidecar flags) on top of the same Cordis Host/UI stack.

## Resolution rules

| Surface | Order |
|---------|-------|
| `dsh` via `desktop-cli` / terminal shim | `DSH_CLI_BIN` then home (`DSH_HOME` / `~/.dsh` / … `node_modules/@deepseek-ai/dsh/lib/bin.js`); never silent ASAR; opt-in `DSH_CLI_ALLOW_BUNDLED=1` |
| Wails Host spawn | `DSH_BIN` override; then packaged/monorepo Host launcher (`host-main` / workspace); then user-home Host install as escape hatch |

## How to try

1. Home CLI present: install `@deepseek-ai/dsh` under `~/.dsh`, run `desktop-cli --version` → home entry + stderr reason.
2. Home CLI missing: empty HOME → friendly missing text (no silent bundled).
3. Wails hybrid with monorepo layout: Host boots from packaged `host-main` (same Cordis Web UI family).
4. Unit: `vitest` `cli-home-resolve` + `desktop-cli`; `go test` in `wails/` (`TestPackagedBeatsUserHome`).

## Acceptance

- Concept docs describe Host as Cordis web-profile UI stack (not a separate mystery runtime).
- CLI home-first verified by vitest.
- Wails packaged Host path still preferred when layout exists (`TestPackagedBeatsUserHome`).
- Push only `origin/feat/wails-user-dsh-home`.
