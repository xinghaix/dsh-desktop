# Wails workspace scripts

Recommended hybrid entry (repo root or dsh-plugin-desktop):

- start:wails / dev:wails / build:wails / package:wails / smoke:wails
- start:host / start:host:node / start:host:electron-as-node
- sync:native-ui:wails (copies lib/native-ui into wails embed tree)
- node scripts/wails-smoke.mjs (non-workflow smoke)

Helper: dsh-plugin-desktop/wails/scripts/run-wails.mjs

Go resolution: DSH_GO_BIN -> PATH go -> ~/sdk/go1.27.0/bin -> ~/go/bin -> /home/box/sdk/go1.27.0/bin

Electron start/dev remain fallback Host+BrowserWindow path. ci.yml stays Electron/Yarn based; wails-smoke.yml is additive when workflow scope allows.
