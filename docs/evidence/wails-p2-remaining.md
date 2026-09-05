# Remaining after P2 push (2026-09-05 Asia/Shanghai)

## Advanced this session (Done or meaningfully improved)

| Item | Status | Notes |
| --- | --- | --- |
| 13 LAN HTTPS announce/status UX | Done (Linux bed) | Multi-line Tools dialog; compact Capabilities line |
| 14 Formal CI wails-smoke | Blocked | gh scopes gist/read:org/repo — no workflow; example kept |
| 15 AppImage package deps + e2e | Done on bed | file(1) dep documented; AppImage/deb/rpm/aur built (gitignored) |
| 16 Release path strategy | Done (docs) | Flip checklist; no product CI flip |
| 17 Recovery checkpoint/uninstall | Partial | Debt InfoDialog + precise Host API gaps; controller still missing |
| 18 Electron LAST-RESORT | Done (docs) | Trigger conditions + pressure-test checklist |
| AuthProxy packaged artifacts | Docs Done | AUTH.md table; packaged AppImage AuthProxy runtime smoke still optional |
| Crash-evidence Control UI UX | Done (Linux bed) | Multi-line status; Reveal folder; Control UI buttons |
| macOS-only | Out-of-bed | Documented in wails-migration.md |

## Still Partial / next actions

1. Workflow-scoped credential → commit `.github/workflows/wails-smoke.yml` from example.
2. Host recovery API: listHealthyCheckpoints / preview+confirm restore+uninstall + generation quiesce.
3. Optional: boot packaged AppImage with Host sidecar and confirm AuthProxy in Help menu.
4. Darwin bed: Dock/notarize/tray template smoke.
5. LAST-RESORT electron-main GUI pressure-test screenshots when feasible.

## N/A on this bed (unchanged)

- system bus sleep/wake
- interactive notify click / hide-to-tray click E2E
