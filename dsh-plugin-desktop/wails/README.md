# DSH Desktop — Wails v3 native shell

Go **1.27** + **Wails v3.0.0-beta.16** shell that replaces Electron's
BrowserWindow / Tray / Menu / Dialog surface over time.

Placement: `dsh-plugin-desktop/wails/` (see `LOCATION.md`). Cordis Host still
boots as a Node sidecar during the hybrid migration — see
`../src/wails-shell-bridge.md` and `../../../docs/wails-migration.md`.

## Tooling (this cloud box)

```bash
export PATH="/home/box/sdk/go1.27.0/bin:/home/box/go/bin:$PATH"
export GOPATH=/home/box/go
go version          # go1.27.0
wails3 version      # v3.0.0-beta.16
```

Requires GTK4 + webkitgtk-6.0 (already installed on the migration box).

## Build

```bash
cd dsh-plugin-desktop/wails

# Fast compile check (embeds frontend/dist)
go build -o bin/dsh-wails-shell .

# Full Wails pipeline (builds frontend via Taskfile when JS toolchain is available)
wails3 build
```

Regenerate TypeScript bindings after changing exported Go service methods:

```bash
wails3 generate bindings -ts -i -b ./...
```

A minimal static control UI is committed under `frontend/dist/` so `go build`
works without Vite. Prefer `frontend/` + Vite when the JS toolchain is usable.

## Run

```bash
# Control UI only
go run .

# Point at an already-running Cordis Host web UI (Electron's loadURL target)
DSH_HOST_URL='http://127.0.0.1:PORT/' go run .
# or:
go run . -host-url 'http://127.0.0.1:PORT/'

# Hybrid sidecar: spawn a host command and wait for readiness
DSH_HOST_URL_FILE=/tmp/dsh-host-url \
DSH_HOST_COMMAND='your-host-boot && node scripts/announce-host-ready.mjs http://127.0.0.1:PORT/' \
  go run .
```

`scripts/announce-host-ready.mjs` prints `DSH_HOST_READY <url>` and optionally
writes `DSH_HOST_URL_FILE`.

## Native APIs used (Wails v3, not v2)

- `application.New` / `app.Window.NewWithOptions` / `window.SetURL`
- `app.SystemTray.New` + tray menu
- `app.NewMenu` / `app.Menu.Set`
- `app.Dialog.OpenFile` / `Info` / `Warning` / `Error`
- Services + `wails3 generate bindings`

## Remaining Electron debt

See `../../../docs/wails-migration.md` for the live checklist.

### Host auto-start

By default the shell auto-starts Cordis Host via the existing desktop start path in sidecar mode. Use -no-host or DSH_HOST_AUTOSTART=0 for control UI only.

## Workspace scripts (preferred hybrid entry)

See package.json scripts build:wails, start:wails, dev:wails, package:wails in dsh-plugin-desktop. Electron start/dev remain the fallback until wails3 package replaces electron-builder for the main path.

