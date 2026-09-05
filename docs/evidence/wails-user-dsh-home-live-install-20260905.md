# Live user `~/.dsh` install + Wails hybrid (2026-09-05)

Timezone: log timestamps UTC; user zone Asia/Shanghai (UTC+8).

Branch: `feat/wails-user-dsh-home` (do not touch `feat/wails3-shell`).

## Install layout (`/home/box/.dsh`)

Standalone-looking Desktop Host home alongside existing Cordis CLI data:

| Path | Role |
|------|------|
| `bin/dsh-desktop` | Executable shim → Node 22 + `realpath(lib)/bin.js` |
| `bin/dsh-plugin-desktop` | Symlink → `dsh-desktop` |
| `lib/` | Symlink → monorepo `dsh-plugin-desktop/lib` (host-main, bin.js, …) |
| `node_modules/` | Symlink → plugin `node_modules` |
| `package.json` | Symlink → plugin package.json |
| `profiles/`, `sessions/`, `settings.yaml`, `storages/` | Pre-existing Cordis CLI user data (kept) |

Notes:

- Shim uses `readlink -f` on `lib` so Node’s main-module guard matches when `lib` is a symlink.
- Do **not** place the plugin’s `cordis.patch.yml` at `~/.dsh/cordis.patch.yml` — Cordis treats that as a home patch layer and duplicates `desktop-shell` when composed with the install patch.
- Happy-path launch unsets `DSH_BIN` / `DSH_HOME` / `DSH_HOST_COMMAND` so discovery is home-scan only.

See also `docs/evidence/wails-user-dsh-home-logs/INSTALL_LAYOUT.txt`.

## Hybrid test (binary from `/tmp`)

```text
cp wails/bin/dsh-wails-shell /tmp/dsh-wails-shell
HOME=/home/box DISPLAY=:5 GTK_A11Y=none /tmp/dsh-wails-shell
```

### Discovery source

```text
Desktop Host via HOME:~/.dsh:bin/dsh-desktop → /home/box/.dsh/bin/dsh-desktop
```

### Host UI

- `[dsh-host] DSH_HOST_READY http://127.0.0.1:43123/…`
- `loaded Cordis Host UI http://127.0.0.1:39039/` (AuthProxy)
- Cordis Host chat / workspace UI rendered (API-key onboarding modal visible)

### Control UI (`-no-host`)

Log line:

```text
Host discovery: host-discover=ok reason=HOME:~/.dsh:bin/dsh-desktop path=/home/box/.dsh/bin/dsh-desktop
```

## Evidence files

| File | What |
|------|------|
| `wails-user-dsh-home-logs/user-dsh-home-live-hybrid.log` | Hybrid launch log |
| `wails-user-dsh-home-logs/user-dsh-home-live-discover-probe.log` | `ProbeHostDiscovery` unit probe |
| `wails-user-dsh-home-logs/user-dsh-home-live-nohost-discover.log` | Control UI `-no-host` log |
| `wails-user-dsh-home-present-hybrid-live-20260905.png` | Host UI screenshot |
| `wails-user-dsh-home-present-hybrid-live-chat-20260905.png` | Host UI (repeat) |
| `wails-user-dsh-home-nohost-discover-live-20260905.png` | Control UI + discovery log |
| Artifacts mirror | `/workspace/artifacts/user-dsh-home-live-*` |

## Blockers / notes

1. First attempt failed after putting plugin `cordis.patch.yml` into `~/.dsh/` (duplicate `desktop-shell`). Fixed by removing that home-level symlink.
2. Electron dist was downloaded on first electron-as-node launch (`[dsh-host] Downloading Electron binary...`); subsequent run used local `node_modules/electron/dist/electron`.
3. Shell binary used as already built @ `wails/bin/dsh-wails-shell` (Go 1.27); copied to `/tmp` for the happy-path run.
