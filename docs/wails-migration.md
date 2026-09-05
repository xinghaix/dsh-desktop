# Electron to Wails v3 migration

Branch: feat/wails3-shell
Shell: dsh-plugin-desktop/wails/

## Primary run path

- start:wails / dev:wails / smoke:wails (repo root scripts)
- start:host (Node Host sidecar)
- node scripts/wails-smoke.mjs
- Electron main.ts is LAST-RESORT (docs/electron-shell-fallback.md)

## Done

- Main webview, tray, menus, dialogs
- Host sidecar auto-start (Node / Electron-as-Node / LAST-RESORT Electron main)
- AuthProxy production path (loopback hardened)
- Aux Recovery/Setup/Profile; prefers lib/native-ui + scheme bridge
- Notifications, export, reveal, terminal, updates (mac/win/linux AppImage download URL)
- File-based crash evidence (active-run + panic dumps; Reveal folder + Control UI status)
- Dock/taskbar attention via Flash + tray tooltip count
- Windows AppUserModelID; Node AES-GCM LAN HTTPS protector
- Host DSH_HOST_LAN_HTTPS announce -> multi-line LAN HTTPS Status + compact Capabilities Status
- Root/workspace wails scripts; scripts/wails-smoke.mjs; docs/wails-ci-smoke.yml.example (workflow push needs workflow scope)
- package:wails Go-binary fallback + AppImage dependency probe (package-deps)
- host-main/host-launcher/node-desktop-runtime tsc --emitDeclarationOnly clean
- Linux bed: tray watcher via haskell-gtk-sni + Node 22 login PATH scripts; sleep/wake documented N/A

## Permanently platform-blocked

- Native webview per-request header hooks (AuthProxy workaround)
- Electron Crashpad/minidumps (file-based substitute only)
- macOS numeric Dock badge / setBadgeCount (Flash + tooltip only)
- Interactive GUI on truly headless Linux (this cloud box has DISPLAY — see docs/wails-linux-smoke.md)
- wails3 package AppImage needs file(1)+FUSE+linuxdeploy; Linux bed produced AppImage 2026-09-05 (preview only; electron-builder still ships)

## Release notes — packaging path (electron-builder vs Wails)

**Shipping / product CI today:** electron-builder remains the **authoritative** desktop
release path (GitHub Actions product workflows, app.asar / platform installers).
Do not announce Wails AppImage/deb as the primary download channel until an explicit
release flip lands.

**Primary developer / hybrid run path today:** Wails + Node Cordis Host
(start:wails / smoke:wails). Electron BrowserWindow / Tray / main.ts is
last-resort Host GUI only.

**Parallel packaging:** package:wails exercises wails3 package when the host has
tooling; otherwise it writes bin/dsh-wails-shell. Linux AppImage dependency details:
docs/wails-package-appimage.md. CI smoke job stays example-only until workflow scope
allows committing the live wails-smoke workflow file.

When writing release notes for a Wails-hybrid milestone:

1. Call out hybrid run path first (Wails shell + Node Host).
2. State clearly that **installers users download are still electron-builder** until flip.
3. Mention AppImage/package:wails as preview/packaging R&D, not the default channel.
4. Link docs/wails-migration.md and dsh-plugin-desktop/docs/electron-shell-fallback.md.

## Release flip checklist (electron-builder → Wails primary)

Do **not** flip product CI casually. Flip only when all of the following are true and
documented in release notes + this file:

1. Live `.github/workflows/wails-smoke.yml` green on linux (and preferably mac/win runners).
2. `wails3 package` produces signed/notarized (or equivalent) installers for each shipping OS.
3. AuthProxy Origin rewrite + CookieJar verified on those packaged artifacts (not only go-build smoke).
4. Recovery Host API owns checkpoint/uninstall generation state (or release notes explicitly
   defer those tabs with operator guidance).
5. AppImage/FUSE/linuxdeploy (or platform equivalents) available on packaging runners.
6. Product download URLs and update checker pointed at Wails artifacts; electron-builder
   retained as emergency LAST-RESORT channel with a documented rollback.
7. Explicit owner sign-off recorded in the PR that changes default CI / download channel.

Until then: electron-builder = authoritative shipping path; Wails = primary hybrid run path
+ parallel packaging R&D.

## macOS-only verification (out of Linux bed)

This cloud Linux bed cannot validate:

- macOS Dock badge / template tray icons / notarization
- MacOptions titlebar / backdrop appearance
- codesign / notarize / Sparkle-adjacent update handoff on Darwin

Track macOS verification on a Darwin host; do not block Linux hybrid P2 on those items.

## Formal CI blocker (workflow scope)

`gh auth status` on the xinghaix push credential (2026-09-05): scopes `gist`, `read:org`, `repo` —
**no `workflow` scope**. Pushing `.github/workflows/wails-smoke.yml` is rejected until a
credential with workflow scope is available. Keep `docs/wails-ci-smoke.yml.example` in sync;
local smoke remains `node scripts/wails-smoke.mjs`.

## Recovery Host↔Wails RPC (transport Done + confirm UX)

Wails aux Recovery is **not** a full port of Electron
DesktopStartupRecoveryWindow (native DesktopDialogWindow chrome), but Host↔Wails
RPC + hybrid confirm flow now exist.

**Landed (2026-09-05):**
- `host-main` / LAST-RESORT `main` keep `DesktopStartupRecoveryController` alive in
  Wails sidecar recovery mode (no immediate dispose/exit).
- Loopback Recovery RPC (`src/wails-recovery-rpc.ts`): announce
  `DSH_HOST_RECOVERY_RPC http://127.0.0.1:PORT/ token=…`
- Endpoints (Bearer token): `GET /v1/health`, `GET /v1/snapshot`,
  `POST /v1/checkpoint/preview|execute|open`, `POST /v1/uninstall/preview|execute`,
  `POST /v1/complete`, `POST /v1/quiesce`, `POST /v1/diagnostics/export`.
- Go `RecoveryRpcClient` + HostSidecar ingest; AuxWindowService prefers RPC for
  checkpoint/uninstall schemes; OpenRecovery injects snapshot when RPC is up
  (and refreshes when RPC announce races after REQUIRED).
- Confirm UX: preview → `/shell-ui/confirm.html` → execute (Cancel clears pending).
- Hybrid quick wins: diagnostic archive zip (RPC/CLI + Help menu), local config reveal, terminal,
  profile creator / switch-profile preferred hint; generation quiesce via `/v1/quiesce` before mutations.
- HostSidecar treats Recovery RPC announce as readiness (`recovery://rpc`) so
  waitForURL does not time out when Host never announces `DSH_HOST_READY`.

| Surface | Wails hybrid today | Still remaining |
| --- | --- | --- |
| Open Recovery window | AuxWindowService.OpenRecovery + native-ui | — |
| Host ask for Recovery | DSH_HOST_RECOVERY_REQUIRED opens aux | — |
| Recovery RPC keep-alive | DSH_HOST_RECOVERY_RPC + controller | — |
| Checkpoint list | snapshot injected when RPC attached | Empty/unavailable when Host never started RPC |
| Checkpoint preview/restore | preview → Confirm dialog → execute | Native DesktopDialogWindow chrome parity |
| Plugin uninstall preview/confirm | preview → Confirm dialog → execute | Same; immutable-target errors via InfoDialog |
| Restart / safe-mode / quit | CompleteRecovery + `/v1/complete` (quiesce then settle) + StopHostSidecar | No drain/idle/cancel-in-flight Host API beyond `quiesceForRecovery` |
| Diagnostics / config / terminal | Diagnostic archive zip (RPC/CLI) + Save/reveal; config reveal; terminal | Darwin SaveFileDialog verify; Crashpad dumps N/A in hybrid |
| Darwin / CI workflow | — | Out of Linux bed; workflow scope blocker |

**Operator expectation:** With Host in recovery keep-alive, Recovery tabs list
checkpoints/bundles from RPC; restore/uninstall require Confirm. Without RPC,
debt InfoDialogs remain for checkpoint/uninstall.

Canonical surface map: dsh-plugin-desktop/src/wails-shell-bridge.md
Evidence: docs/evidence/wails-p2-recovery-rpc-20260905.md,
docs/evidence/wails-p2-recovery-rpc-ux-20260905.md,
docs/evidence/wails-p2-diagnostics-quiesce-20260905.md
