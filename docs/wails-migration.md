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

## Recovery controller debt

Wails aux Recovery is **not** a full port of Electron
DesktopStartupRecoveryController (src/startup-recovery-controller.ts +
src/startup-recovery-window.ts).

| Surface | Wails hybrid today | Still Electron / Host debt |
| --- | --- | --- |
| Open Recovery window | AuxWindowService.OpenRecovery + native-ui /aux HTML | — |
| Host ask for Recovery | DSH_HOST_RECOVERY_REQUIRED opens aux | — |
| Restart / safe-mode / quit / profiles / control | CompleteRecovery shell actions | No generation quiesce contract |
| Checkpoint list / restore preview / confirm | Not wired | Controller preview TTL + restore slots |
| Plugin uninstall preview / confirm | Not wired | Controller uninstall preview + immutable-target rules |
| Pre-Host generation authority | Absent in Go shell | DesktopStartupRecoveryController binds one generation id |
| Destructive confirm dialogs | Status dialogs only | Electron DesktopDialogWindow confirm path |

**Operator expectation:** Recovery Assistant in Wails can surface failure text and
relaunch/switch-profile affordances. Checkpoint rollback and plugin uninstall from
Recovery remain Electron-main / controller debt until a Host-facing recovery API
owns that state for the sidecar. Do not claim parity with Electron Recovery tabs for
uninstall/checkpoint in release notes.

Canonical surface map: dsh-plugin-desktop/src/wails-shell-bridge.md
