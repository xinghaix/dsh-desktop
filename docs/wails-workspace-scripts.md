# Wails workspace scripts

Primary hybrid entry scripts are defined on the dsh-plugin-desktop package.json:

- build:wails
- start:wails
- dev:wails
- package:wails

Helper module: dsh-plugin-desktop/wails/scripts/run-wails.mjs

Electron start/dev remain the fallback Host+BrowserWindow path. CI packaging still uses electron-builder under dsh-plugin-desktop/scripts until wails3 package replaces it.
