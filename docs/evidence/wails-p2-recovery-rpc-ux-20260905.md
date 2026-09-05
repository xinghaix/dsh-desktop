# Wails Recovery RPC UX polish (2026-09-05 Asia/Shanghai)

Branch: `feat/wails3-shell` (on tip including `19fb335473`).

## Goal

Electron-style **preview → user confirm → execute** for checkpoint restore /
plugin uninstall over Host Recovery RPC, plus small Recovery action quick wins,
then hybrid smoke.

## Done

| Item | Status |
| --- | --- |
| Confirm dialog (`/shell-ui/confirm.html`) + `OpenConfirmDialog` / `CompleteConfirmDialog` | Done |
| Checkpoint / uninstall preview stores pending; execute only after confirm | Done |
| Cancel clears pending; no execute | Done (unit tested) |
| Scheme bridge passes `id` / switch-profile `name`; click capture handler | Done |
| Refresh Recovery window when `DSH_HOST_RECOVERY_RPC` attaches (race with REQUIRED) | Done |
| Diagnostics → reveal crash-evidence folder (best-effort) | Done |
| Config buttons → reveal local settings.yaml / cordis.patch.yml / package.json / profile dir | Done |
| Terminal / profile creator / switch-profile hybrid handlers | Done |
| `configurationAvailable` + `terminalAvailable` + `profileActionToken` in Recovery state | Done |
| Unit tests (`aux_recovery_confirm_test.go`) | Done |
| Hybrid smoke: announce + health/snapshot curl + GUI screenshots | Done |

### Confirm flow

1. Native-ui `preview-checkpoint` / `preview-uninstall` → `CompleteRecovery`
2. Go calls Host RPC **preview only**
3. Opens webview confirm (Linux-safe; GTK Question can be silent on this bed)
4. User Confirm → `CompleteConfirmDialog("confirm")` → RPC **execute**
5. User Cancel → clear pending; no execute

### Smoke (Linux bed, DISPLAY=:5)

Command shape:

```bash
export DSH_HOST_COMMAND="cd dsh-plugin-desktop && export DSH_WAILS_HOST_SIDECAR=1; \
  node lib/host-main.js --dsh-wails-host-sidecar --dsh-desktop-recovery"
./bin/dsh-wails-shell
```

Observed:

- `[dsh-host] DSH_HOST_RECOVERY_REQUIRED …`
- `[dsh-host] DSH_HOST_RECOVERY_RPC http://127.0.0.1:<port>/ token=…`
- `curl` `GET /v1/health` → `ok=true`, `hasController=true`
- `curl` `GET /v1/snapshot` → `profileName=desktop`, 2 core bundles, 3 available checkpoints, `controller=true`
- Recovery UI: **Current Profile: desktop**; Plugin management lists `@deepseek-ai/dsh-base` / `@deepseek-ai/dsh-web-app`; Rollback shows Slot 1 (2.0.5 / 2 plugins / 5 files / 992 B); Diagnostics shows Configuration files row

Artifacts:

- `/workspace/artifacts/wails-recovery-rpc-ux-20260905.png`
- `/workspace/artifacts/wails-recovery-rpc-tabs-20260905.png` (Plugin management RPC bundles)
- `/workspace/artifacts/wails-recovery-rpc-checkpoints-20260905.png` (Rollback slots)
- `/workspace/artifacts/wails-recovery-rpc-diagnostics-20260905.png` (config quick-win UI)

Also copied under `docs/evidence/` where applicable.

## Remaining backlog

1. ~~Full Electron diagnostic archive export (zip)~~ → Done in `wails-p2-diagnostics-quiesce-20260905.md` (RPC/CLI + UI path).
2. ~~Generation quiesce beyond StopHostSidecar~~ → Best-effort `/v1/quiesce` + complete/execute wiring; no finer Host drain/idle API.
3. Darwin verification of confirm + Recovery RPC + SaveFileDialog export.
4. Live `.github/workflows/wails-smoke.yml` (workflow credential scope — out of this pass).
5. Electron LAST-RESORT BrowserWindow Recovery parity when not Wails-light (out of scope).
6. Interactive GUI pressure-test of Confirm/Cancel click path on every bed (unit-tested; xdotool click hit-or-miss on WebKit here).
7. `open-profile-patch` / manifest when files do not exist yet → informational dialog (already); Host-authoritative paths when recovery home differs.

## Non-goals / do not regress

- AuthProxy Origin rewrite, shell-ui embed, Recovery `notice:null` fix, AppImage path, RPC transport itself.
