# Recovery checkpoint / uninstall Host API investigation (2026-09-05 Asia/Shanghai)

Branch tip base: `ba02b8d6d3` on `feat/wails3-shell`.

## Question

Can the Wails Recovery window call existing Host/Cordis APIs for checkpoint list/preview/confirm and plugin uninstall + generation quiesce?

## Finding

**Blocked on missing Host↔Wails transport — not missing TypeScript types.**

| Layer | Status |
| --- | --- |
| `DesktopStartupRecoveryController` (`src/startup-recovery-controller.ts`) | Exists in-process |
| `DesktopProfileCheckpoint` (`src/profile-checkpoint.ts`) | Exists (`listSlots`, restore, …) |
| Cordis HTTP / Host service endpoints for recovery | **None** |
| Stdout announce protocol (`DSH_HOST_*`) | Ready / auth / LAN / recovery-required only — **no RPC** |
| Wails Go shell ownership of generation | Absent |

### Real controller call surface (Node/Electron Host)

- `snapshot()` → `DesktopStartupRecoverySnapshot` (`checkpoints[]`, `bundles[]`)
- `previewCheckpointRestore(slotId)` / `executeCheckpointRestore(previewId)`
- `previewUninstall(bundleId)` / `executeUninstall(previewId)`
- `openCheckpoint(slotId)`
- Generation assert via `currentGeneration()` + dispose

Debt InfoDialog previously named fictional `listHealthyCheckpoints` / `previewPluginUninstall`; updated to the real method names + transport gap.

### Wails Host path today (`host-main.ts`)

On recovery for Wails sidecar:

1. May construct `DesktopStartupRecoveryController`
2. Announces `DSH_HOST_RECOVERY_REQUIRED …`
3. **Disposes** the controller and **exits**
4. Go `AuxWindowService.OpenRecovery` shows UI; checkpoint/uninstall schemes → debt InfoDialog

## What would unblock

1. Host keep-alive in recovery mode (do not dispose/exit immediately)
2. Request/response protocol (stdout lines or loopback HTTP) exposing the controller methods above
3. Wire Recovery native-ui snapshot + confirm to that RPC
4. Generation quiesce on Wails `CompleteRecovery` restart/safe-mode

Until then: keep debt InfoDialogs; do not claim Recovery tab parity in release notes.

See also: `docs/wails-migration.md` § Recovery controller debt; `src/wails-shell-bridge.md`.

## Follow-up

Transport landed: see `docs/evidence/wails-p2-recovery-rpc-20260905.md`.
