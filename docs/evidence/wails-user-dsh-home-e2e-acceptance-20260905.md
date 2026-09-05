# E2E + abnormal acceptance: feat/wails-user-dsh-home (2026-09-05)

Timezone: log timestamps below are **UTC**; user local **Asia/Shanghai (UTC+8)**.

- Branch: `feat/wails-user-dsh-home`
- Base commit at start: `953b55e5f6`
- Tip (auto-relaunch): `1cebf9722f`
- Evidence dir: `docs/evidence/wails-user-dsh-home-e2e-20260905/`
- Binary: `/tmp/dsh-wails-shell` (copy of `dsh-plugin-desktop/wails/bin/dsh-wails-shell`), DISPLAY=:5, GTK_A11Y=none
- Tooling: Go 1.27 `/home/box/sdk/go1.27.0/bin`, Node 22 `/home/box/sdk/node-v22.19.0-linux-x64/bin`

## UX fixes in this pass

| Change | Purpose |
|--------|---------|
| `/shell-ui/host-error.html` | Friendly in-app page for abnormal Host failures (kind badge + steps + detail + Retry / Choose / Install / Recovery) |
| `ShellService.ShowHostFailurePage` / `ShowHostErrorPage` | Route missing → install page; other failures → host-error with classified kind |
| `HostSidecar.OnUnexpectedExit` + `HandleUnexpectedHostExit` | Mid-run Host exit → backoff auto-relaunch (1–3), then host-error (`crash`) |
| `host-restarting.html` | Brief “Restarting Host…” status during auto-relaunch |
| `ExpectHostExit` / `Stop` / `Quit` / Recovery quit | Intentional stop suppresses auto-relaunch |
| `classifyHostFailure` | Maps timeout / port-bind / crash / invalid-home / not-usable / profile / start-failed |
| Invalid `DSH_HOME` messaging | `friendlyInvalidHomeMessage` when env/chosen root missing or empty of Host entries |
| Stdout hint capture | Surfaces `EADDRINUSE` / profile-create lines into timeout error text for classification |

## Happy-path table

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| A | Missing user install → install-help page (`host-install.html`), no packaged Host | **PASS** | `A-missing-install.log` / `.png` (`/shell-ui/host-install.html`) |
| B | Present `~/.dsh` Host → discover + `DSH_HOST_READY` + Cordis UI | **PASS** | `B-present-install.log` / `.png`, `B-present-host-chat.png` |
| C | Monorepo layout + empty home: packaged Host off by default | **PASS** | `C-no-packaged-default.log` / `.png` (`host-install.html`, “packaged Host disabled by default”) |
| U | `go test ./...` in `wails/` | **PASS** | `go-test.log` |
| V | vitest `cli-home-resolve` + `desktop-cli` | **PASS** | `vitest-cli-home-resolve.log` (13 tests) |

## 异常场景 / Abnormal scenarios

| # | Scenario | Result | Evidence |
|---|----------|--------|----------|
| D | Invalid / missing `DSH_HOME` path | **PASS** | `D-invalid-dsh-home.log` / `.png` — kind `invalid-home`, path `/tmp/e2e-D-does-not-exist` |
| E | Path exists but not a usable Host | **PASS** | `E-not-usable-host.log` / `.png` — kind `not-usable`, empty `/tmp/e2e-E-empty-dsh` |
| F | Host start / ready timeout | **PASS** | `F-ready-timeout.log` / `.png` — kind `timeout`, `DSH_HOST_READY_TIMEOUT=2s`, fake Host never announces READY |
| G | Host crash / exit mid-run → friendly page (auto-relaunch disabled) | **PASS** | `G-host-crash.log` / `.png` — `DSH_HOST_RELAUNCH_MAX=0`; READY then `exit 1`; kind `crash` |
| K1 | Crash storm → 3 auto-relaunches then host-error | **PASS** | `K-relaunch-exhausted.log` / `.png`, `K-relaunch-restarting.png` |
| K2 | Single crash → auto-relaunch recovers (stable Host) | **PASS** | `K-relaunch-recover.log` / `.png` |
| K3 | Intentional stop / suppress does not relaunch | **PASS** | `K-intentional-stop.log` (unit) + `K-intentional-stop-live.log` |
| H | Port bind failure (simulated EADDRINUSE) | **PASS** | `H-port-bind.log` / `.png` — kind `port-bind`, stdout `EADDRINUSE :::43124` |
| I1 | Profile fallback to `web` when no local profiles | **PASS** | Unit: `I-profile-web-fallback.log` (`TestE2EProfileFallbackWeb`); Live boot: `I-profile-web-live.log` / `.png` (isolated HOME/XDG + `DSH_BIN`) |
| I2 | Failure to create web profile (simulated) | **PASS** | `I-profile-create-fail.log` / `.png` — kind `profile`; `TestClassifyProfileCreateFailure` |
| J | CLI missing-home friendly error (no silent bundled) | **PASS** | `J-cli-missing-home.log`, `J-cli-missing-home-message.log` (“No user-installed dsh CLI…”, `DSH_CLI_ALLOW_BUNDLED=(off)`) |

## Notes

- Auto-relaunch defaults: max **3**, backoff **1s,2s,3s**, stable reset after **45s** uptime (`DSH_HOST_RELAUNCH_*` env overrides).
- Intentional `Quit` / `StopHostSidecar` / Recovery quit call `ExpectHostExit` so Host exit does not loop-restart.
- Discovery order unchanged: `DSH_BIN` → `DSH_HOME` / chosen → home roots → `~/.local/bin` → PATH; packaged only with `DSH_ALLOW_PACKAGED_HOST=1`.
- A/B/C from earlier partial run reused where still valid; C screenshot refreshed; F–I re-run after UX rebuild.
- Profile create failure is safely simulated via `DSH_HOST_COMMAND` printing `failed to create web profile` (not a real Cordis materializer fault injection).

## How to reproduce (abnormal)

```bash
export PATH="/home/box/sdk/go1.27.0/bin:/home/box/sdk/node-v22.19.0-linux-x64/bin:$PATH"
export DISPLAY=:5 GTK_A11Y=none
cp dsh-plugin-desktop/wails/bin/dsh-wails-shell /tmp/dsh-wails-shell

# D invalid home
HOME=/tmp/empty XDG_CONFIG_HOME=/tmp/empty-xdg/config DSH_HOME=/no/such /tmp/dsh-wails-shell

# F timeout
# put a sleep-only ~/.dsh/bin/dsh-desktop under a fake HOME; DSH_HOST_READY_TIMEOUT=2s

# G crash page only (no auto-relaunch):
DSH_HOST_RELAUNCH_MAX=0  # Host prints DSH_HOST_READY then exit 1

# K exhaust: same crashing Host, default max=3, short backoff
DSH_HOST_RELAUNCH_BACKOFF=200ms,200ms,200ms

# K recover: Host crashes once then stays up on next start
# H port: Host prints EADDRINUSE then exit 1
```
