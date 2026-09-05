# Wails user dsh home + plugin/tool boundary (2026-09-05)

Branch: `feat/wails-user-dsh-home`

## Product rules

1. **No packaged Host by default.** Wails discovers/attaches only a **user-installed** `dsh` Host (`DSH_BIN` → `DSH_HOME` / persisted chosen dir → `~/.dsh`|`~/dsh`|XDG → `~/.local/bin` → PATH). Packaged/monorepo `host-main` requires explicit `DSH_ALLOW_PACKAGED_HOST=1` (dev only).
2. **Missing install → install-help page** (`/shell-ui/host-install.html`) with checked paths, install steps, Choose directory… (persists `user-dsh-home` under Desktop userData and sets `DSH_HOME`).
3. **Profile:** prefer an existing user profile; otherwise use/create **`web`** (`dsh --profile web`).
4. **Control/connect dsh** capabilities that historically lived in packaged `host-main` should be delivered as **tools/plugins on the user install**, not welded into a bundled Host.

## Interface boundary (short-term)

| Layer | Owns |
|-------|------|
| Wails shell | Native window, discovery, install-help UX, sidecar spawn/attach, auth proxy, recovery windows |
| User `dsh` Host | Cordis Web UI, profiles, plugin loader, desktop bridge announce when sidecar flag present |
| Future desktop control plugins | Actions that mutate/control dsh (plugin install, profile ops beyond shell hints) — run against user install |

Do **not** add new product boot paths that spawn monorepo `dsh-plugin-desktop/lib/host-main.js` without `DSH_ALLOW_PACKAGED_HOST=1`.

## CLI

`desktop-cli` / `cli-home-resolve`: home-first (`DSH_CLI_BIN` → `DSH_HOME` / `~/.dsh` …). No silent ASAR. Opt-in `DSH_CLI_ALLOW_BUNDLED=1` only.

## Verify

- Empty home + no `DSH_*` → install page; no monorepo host spawn
- `~/.dsh` Host present → Wails uses it
- CLI respects home / `DSH_HOME`
- `go test` in `wails/` + vitest `cli-home-resolve`
