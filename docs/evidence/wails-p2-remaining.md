# Remaining after P2 continuation (2026-09-05 Asia/Shanghai)

## Advanced this session (Done or meaningfully improved)

| Item | Status | Notes |
| --- | --- | --- |
| 13 LAN HTTPS announce/status UX | Done (Linux bed) | Multi-line Tools dialog; compact Capabilities line |
| 14 Formal CI wails-smoke | Blocked | gh scopes gist/read:org/repo — no workflow; example kept |
| 15 AppImage package deps + e2e | Done on bed | file(1) dep documented; packages built (gitignored) |
| 16 Release path strategy | Done (docs) | Flip checklist; no product CI flip |
| 17 Recovery checkpoint/uninstall | Partial→documented | Controller exists in-process; Host↔Wails RPC missing; debt InfoDialog names fixed |
| 18 Electron LAST-RESORT | Done (docs) | Trigger conditions + pressure-test checklist |
| AuthProxy packaged AppImage smoke | Done on bed | extract-and-run + Host sidecar; loopback AuthProxy load proven |
| Crash-evidence Control UI UX | Done (Linux bed) | Multi-line status; Reveal folder; Control UI buttons |
| macOS-only | Out-of-bed | Documented in wails-migration.md |

## Still Partial / next actions

1. Workflow-scoped credential → commit `.github/workflows/wails-smoke.yml` from example.
2. **Host↔Wails recovery RPC** (keep controller alive): `snapshot` / `previewCheckpointRestore` / `executeCheckpointRestore` / `previewUninstall` / `executeUninstall` + generation quiesce — then wire Recovery UI.
3. Darwin bed: Dock/notarize/tray template smoke.
4. LAST-RESORT electron-main GUI pressure-test screenshots when feasible.
5. Optional: Help-menu AuthProxy status click path inside packaged AppImage (load already proven via log).

## N/A on this bed (unchanged)

- system bus sleep/wake
- interactive notify click / hide-to-tray click E2E
