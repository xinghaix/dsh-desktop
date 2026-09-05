# Wails Recovery Host↔RPC transport (2026-09-05 Asia/Shanghai)

Branch: `feat/wails3-shell` (builds on `00659b7394`).

## Goal

Move Recovery checkpoint/uninstall past **Partial** by landing Host keep-alive +
loopback RPC the Go shell can call.

## Done

| Item | Status |
| --- | --- |
| Host keep-alive in Wails recovery (no dispose/exit before shell RPC) | Done (`host-main.ts`, LAST-RESORT `main.ts`) |
| Loopback Recovery RPC server | Done (`src/wails-recovery-rpc.ts`) |
| Announce `DSH_HOST_RECOVERY_RPC` | Done |
| Go client + HostSidecar ingest | Done (`wails/recovery_rpc.go`, `hostsidecar.go`) |
| AuxWindowService checkpoint/uninstall → RPC | Done (debt only when RPC absent) |
| OpenRecovery snapshot inject | Done when RPC attached |
| Automated tests | Done: `tests/wails-recovery-rpc.spec.ts`, `wails/recovery_rpc_test.go`, updated debt/native-ui tests |

### RPC surface

```
DSH_HOST_RECOVERY_RPC http://127.0.0.1:<port>/ token=<bearer>

GET  /v1/health
GET  /v1/snapshot
POST /v1/checkpoint/preview   { "slotId" }
POST /v1/checkpoint/execute   { "previewId" }
POST /v1/checkpoint/open      { "slotId" }
POST /v1/uninstall/preview    { "bundleId" }
POST /v1/uninstall/execute    { "previewId" }
POST /v1/complete             { "action": "restart"|"safe-mode"|"quit" }
```

Authorization: `Authorization: Bearer <token>` (loopback only).

Structural snapshot without controller: `{ profileName:"", bundles:[], checkpoints:[], controller:false }`.

## Remaining

See follow-up polish: `docs/evidence/wails-p2-recovery-rpc-ux-20260905.md`.

- Generation quiesce contract beyond StopHostSidecar + `/v1/complete`.
- Darwin verification; live `.github/workflows/wails-smoke.yml` (workflow scope).
- Electron LAST-RESORT GUI parity for Recovery window (BrowserWindow path unchanged when not Wails-light).
- Full Electron diagnostic archive export (hybrid reveals crash-evidence only).

## Non-goals / do not regress

- AuthProxy Origin rewrite, shell-ui embed, Recovery `notice:null` fix, AppImage path.
