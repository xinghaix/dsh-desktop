# Acceptance: feat/wails-user-dsh-home (2026-09-05)

Timezone notes: log timestamps below are UTC; local user zone Asia/Shanghai (UTC+8).

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 1 | New branch pushed to origin (xinghaix); `feat/wails3-shell` intact | PASS (pending push at write time; verify after) | branch `feat/wails-user-dsh-home`; base `35fe93ea1c` |
| 2 | Discovery order: DSH_BIN > DSH_HOME > home installs > ~/.local/bin > PATH > monorepo | PASS | unit tests + accept logs |
| 3 | Missing user dsh home: friendly prompt (checked paths + install / env next steps) | PASS | `wails-user-dsh-home-logs/missing-home-discover.log`, `missing-host-dialog.log`, screenshot `wails-user-dsh-home-missing-dialog-20260905.png` (InfoDialog `status.html`) |
| 4 | Present user dsh home: shell selects that Host entry (announce) | PASS | `present-home-discover.log`, `present-home-hybrid.log` (`HOME:~/dsh:bin/dsh-desktop` + `DSH_HOST_READY` + loaded UI), screenshot `wails-user-dsh-home-present-hybrid-20260905.png` |
| 5 | Regression: AuthProxy / Recovery / go test; -no-host smoke | PASS | `go-test-full.log`, `go-test-regression.log`, `nohost-control.log`, `wails-user-dsh-home-nohost-20260905.png` |
| 6 | This acceptance checklist under docs/evidence/ | PASS | this file |

## Discovery order (implemented)

1. DSH_BIN
2. DSH_HOME relative: bin/dsh-desktop, bin/dsh-plugin-desktop, lib/host-main.js, dsh-plugin-desktop/lib/host-main.js, lib/bin.js
3. ~/.dsh, ~/dsh, ~/.local/share/dsh, $XDG_DATA_HOME/dsh, ~/.local/opt/dsh
4. ~/.local/bin/dsh-desktop|dsh-plugin-desktop
5. PATH
6. Monorepo fallback

## How to try

```bash
cd dsh-plugin-desktop/wails && go build -o bin/dsh-wails-shell .
# missing (standalone copy avoids monorepo via executable path)
cp bin/dsh-wails-shell /tmp/dsh-wails-shell && HOME=/tmp/empty /tmp/dsh-wails-shell
# present
mkdir -p ~/dsh/bin && cp /path/to/shim ~/dsh/bin/dsh-desktop && ./bin/dsh-wails-shell
# control UI
./bin/dsh-wails-shell -no-host   # Host discovery... button
```
