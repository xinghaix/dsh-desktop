# Wails user dsh home-first (2026-09-05)

Branch: feat/wails-user-dsh-home

## Discovery order
1. DSH_BIN
2. DSH_HOME (bin/dsh-desktop, lib/host-main.js, ...)
3. ~/.dsh, ~/dsh, ~/.local/share/dsh, XDG, ~/.local/opt/dsh
4. ~/.local/bin
5. PATH dsh-desktop / dsh-plugin-desktop
6. Monorepo fallback

## UX
Startup log ProbeHostDiscovery; InfoDialog on miss; Control UI Host discovery button.

## Tests
hostbootstrap_test.go HOME/DSH_HOME fixtures; go test .

## Acceptance

See wails-user-dsh-home-acceptance-20260905.md
