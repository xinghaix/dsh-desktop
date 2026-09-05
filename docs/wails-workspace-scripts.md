# Wails workspace scripts

Primary hybrid entry scripts are defined on the dsh-plugin-desktop package.json:

- build:wails
- start:wails
- dev:wails
- package:wails
- smoke:wails

Helper module: dsh-plugin-desktop/wails/scripts/run-wails.mjs

Go toolchain resolution order:

1. `DSH_GO_BIN` (directory containing the `go` binary)
2. `which go` / current PATH
3. Common local SDK layouts (`~/sdk/go1.27.0/bin`, `~/go/bin`, `/usr/local/go/bin`)
4. Cloud-box fallback `/home/box/sdk/go1.27.0/bin` only if present

`package:wails` runs `wails3 package` when available; otherwise it writes
`bin/dsh-wails-shell` via `go build` and exits successfully so CI/local verify
can still produce an artifact. Full platform installers still need a host with
`wails3` + packaging deps.

Electron start/dev remain the fallback Host+BrowserWindow path. Existing
Electron CI (`ci.yml`) is unchanged; `wails-smoke.yml` adds a Go build/test gate.
