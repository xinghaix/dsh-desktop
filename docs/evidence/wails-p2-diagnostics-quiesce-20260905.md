# Wails Recovery diagnostics zip + generation quiesce (2026-09-05 Asia/Shanghai)

Branch: `feat/wails3-shell` (on tip including prior Recovery RPC UX).

## Goals

1. Hybrid-capable **diagnostic archive zip** (reuse Host `exportDesktopDiagnostics` / Electron format).
2. Finer **generation quiesce** where Host APIs exist; otherwise document the gap and improve best-effort ordering.

## Done

### Diagnostic archive zip

| Layer | Change |
| --- | --- |
| Host Recovery RPC | `POST /v1/diagnostics/export` → `{ ok, path }` via `exportDesktopDiagnostics` (same worker/format as `--export-diagnostics`) |
| host-main / LAST-RESORT main | Wire `exportDiagnostics` callback into Recovery RPC keep-alive |
| Go `RecoveryRpcClient` | `ExportDiagnostics` |
| Go `CapabilitiesService` | `ExportDiagnosticArchive(offerSaveDialog)` — prefer RPC, else `node lib/host-main.js --export-diagnostics`; optional SaveFileDialog copy; reveal final path |
| Recovery UI | `export-diagnostics` / `save-diagnostics` / `show-diagnostics` → archive zip + success path InfoDialog (crash-evidence reveal only as export-failure fallback) |
| Help menu | **Export Diagnostic Archive…** |

### Generation quiesce

| Layer | Change |
| --- | --- |
| Host API reality | **Only** `DesktopStartupGeneration.quiesceForRecovery()` exists (dispose Cordis Host fiber + timeout). **No** drain-generations / wait-for-idle / cancel-in-flight Host surface. |
| Recovery RPC | `POST /v1/quiesce` → `{ ok, detail }`; `/v1/complete` runs quiesce before settle and returns `quiesce` in body |
| Go | `Quiesce` client; `quiesceHostBestEffort` before checkpoint/uninstall execute; CompleteRecovery uses longer timeout so Host can quiesce before `StopHostSidecar` |

## Smoke (Linux bed)

### CLI zip (proves Host format path)

```bash
node dsh-plugin-desktop/lib/host-main.js --export-diagnostics
# → /home/box/.config/DSH Desktop/diagnostics/diagnostics-<ts>-<uuid>.zip
```

Observed archive entries include `system-info.txt`, logs, `lifecycle-events/*`, `crash-evidence/active-run.json` (Electron Crashpad dumps absent in hybrid — expected).

### Unit / integration

- vitest `tests/wails-recovery-rpc.spec.ts` — 8 passed (export 200/501, quiesce + complete).
- `go test ./...` in `dsh-plugin-desktop/wails` — ok (RPC client export/quiesce, copy helper, debt copy, quiesce helper).

### GUI limits

- SaveFileDialog / Reveal may be silent or limited on this WebKit/Linux bed; export still writes under `<userData>/diagnostics/` and surfaces the path in InfoDialog / Help status.
- Interactive Confirm/Cancel click pressure-test remains hit-or-miss with xdotool (unchanged).

## Partial / remaining

1. Darwin verification of SaveFileDialog + Help export + Recovery diagnostics.
2. Live `.github/workflows/wails-smoke.yml` (workflow credential scope).
3. Optional: surface quiesce detail line in CompleteRecovery restart InfoDialog (execute paths already include it).
4. Electron LAST-RESORT BrowserWindow Recovery chrome parity (out of scope).

## Non-goals / do not regress

AuthProxy Origin rewrite, shell-ui embed, Recovery `notice:null`, Recovery RPC+confirm UX, AppImage path, P1 SNI/Node22/tsc.
