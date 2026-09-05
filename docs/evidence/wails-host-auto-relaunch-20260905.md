# Wails Host auto-relaunch after crash (2026-09-05)

Timezone: log timestamps **UTC**; user local **Asia/Shanghai (UTC+8)**.

- Branch: `feat/wails-user-dsh-home`
- Scope: when the user-installed Cordis Host sidecar exits unexpectedly after READY, the Wails shell auto-restarts it with backoff; intentional Quit/Stop/Recovery quit do not.

## Behavior

| Case | Result |
|------|--------|
| Unexpected exit after READY | Show `/shell-ui/host-restarting.html`, wait backoff, `StartHostSidecar` again |
| Max attempts (default 3) exhausted | Show existing `/shell-ui/host-error.html` (`crash`) with manual Retry |
| Host stayed READY ≥ stable window (default 45s) | Crash-storm counter resets |
| `StopHostSidecar` / `Quit` / Recovery quit | `ExpectHostExit` — no auto-relaunch |
| Profile switch / Recovery restart | Intentional Stop then Start (suppress cleared on Start) |

Env overrides: `DSH_HOST_RELAUNCH_MAX`, `DSH_HOST_RELAUNCH_STABLE`, `DSH_HOST_RELAUNCH_BACKOFF`.

## Evidence

See `docs/evidence/wails-user-dsh-home-e2e-20260905/`:

- `K-relaunch-restarting.log` / `.png` — “Restarting Host…”
- `K-relaunch-exhausted.log` / `.png` — 3/3 then host-error
- `K-relaunch-recover.log` / `.png` — single crash recovers
- `K-intentional-stop.log` — unit suppress
- `go-test-relaunch.log` — `go test ./...` PASS

## How to try

```bash
export PATH="/home/box/sdk/go1.27.0/bin:/home/box/sdk/node-v22.19.0-linux-x64/bin:$PATH"
export DISPLAY=:5 GTK_A11Y=none
cd dsh-plugin-desktop/wails && go build -o bin/dsh-wails-shell .
cp bin/dsh-wails-shell /tmp/dsh-wails-shell

# Fake crashing Host under ~/.dsh/bin/dsh-desktop:
#   echo DSH_HOST_READY …; sleep 0.3; exit 1
DSH_HOST_RELAUNCH_BACKOFF=200ms,200ms,200ms /tmp/dsh-wails-shell
# → restarting page, then host-error after 3 attempts

# Disable auto-relaunch (immediate crash page):
DSH_HOST_RELAUNCH_MAX=0 /tmp/dsh-wails-shell
```
