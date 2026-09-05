# Wails Linux smoke checklist (non-macOS regression bed)

Use this cloud Linux host (`DISPLAY` + WebKitGTK) to re-verify **cross-platform** Wails hybrid behavior.
Do **not** treat this bed as a substitute for macOS-only surfaces.

Related: `docs/wails-migration.md`, `docs/wails-workspace-scripts.md`, `dsh-plugin-desktop/wails/README.md`.

## Box prerequisites

```bash
export PATH="/home/box/sdk/node-v22.19.0-linux-x64/bin:/home/box/bin:/home/box/sdk/go1.27.0/bin:/home/box/go/bin:$PATH"
export GOPATH=/home/box/go
export DISPLAY=:8   # or current X display
# Host sidecar is spawned via `bash -lc`, so login PATH must resolve Node >= 22
# (engines: ^22.19.0 || >=24). Symlink works: ~/bin/node -> sdk node 22.
eval "$(dbus-launch --sh-syntax)"
# optional: notification daemon for Notify / libnotify
pgrep -x dunst >/dev/null || dunst &
```

Minimal packages: `scrot`, `libnotify-bin`, `dunst`, `dbus-x11`, `haskell-gtk-sni-tray-utils`, `ayatana-indicator-application`, `libayatana-appindicator3-1`.
Tray: start `gtk-sni-tray-standalone -w` on the same session bus (or `source scripts/linux-smoke-env.sh`). Apt install succeeded on this bed.

### Node 22 on bash -lc

See profile.d hook zz-dsh-node22.sh and home bin shims; helper scripts/linux-smoke-env.sh.

### System D-Bus sleep-wake blocked

No systemd init and no run/dbus system bus on this bed; session bus OK; login1 sleep-wake N/A.


## Build / unit smoke

From repo root on `feat/wails3-shell`:

1. `corepack prepare` / activate packageManager from root `package.json`
2. Package install with immutable lockfile
3. `yarn workspace dsh-plugin-desktop build` — need `lib/host-main.js` + `lib/bin.js`
   - Note: `tsdown` optional peer `unrun` must resolve; root engines want Node 22+
   - `tsc --emitDeclarationOnly` may still fail on in-progress host-main types; JS emit is enough for shell smoke
4. `yarn smoke:wails` (go test + go build) and/or `yarn build:wails`
5. Binary: `dsh-plugin-desktop/wails/bin/dsh-wails-shell`

## Cross-platform GUI checks (Linux bed)

Run with screenshots under `/workspace/artifacts/` or `docs/evidence/`.

| # | Check | How | Pass signal |
|---|--------|-----|-------------|
| 1 | Shell window (`-no-host`) | `./bin/dsh-wails-shell -no-host` | Control UI + menus File/View/Tools/Help |
| 2 | Embedded control UI | Buttons visible | Load Host URL / Start Host sidecar / native-ui entries |
| 3 | Hybrid Host sidecar | `yarn start:wails` with Node 22 on login PATH | Log: `DSH_HOST_READY` + `loaded Cordis Host UI` |
| 4 | Node Host without Electron GUI | `yarn start:host:node` | `Electron-light GUI skipped=true` |
| 5 | Menus | File / View / Tools / Help | Actions open / no crash |
| 6 | Native-ui aux | Setup Wizard / Profiles / Recovery | Windows open via scheme/native-ui |
| 7 | Notify | Control UI **Notify** (needs dunst/session bus) | Desktop notification appears |
| 8 | Dialogs | Open Directory / About | Dialog appears |
| 9 | Terminal entry | **Terminal** | Terminal path invoked or graceful error |
| 10 | Export | **Export...** | Export flow starts or errors cleanly |
| 11 | Updates path | **Updates** | Linux AppImage update check path (network may fail) |
| 12 | Crash evidence | Kill -9 shell mid-run; relaunch | `crash-evidence` unclean-exit marker logged |
| 13 | Hide to tray | **Hide to tray** | Works with gtk-sni-tray-standalone -w on same session bus (*Partial* without StatusNotifierWatcher) |
| 14 | AuthProxy / LAN HTTPS | Host announce; Tools LAN HTTPS Status; Help Capabilities Status | Auth header + `DSH_HOST_LAN_HTTPS`; status lan-https=announced |
| 15 | Sleep/wake | login1 PrepareForSleep | **N/A** (no system D-Bus on this bed) |

### Screenshot commands

```bash
scrot -z /workspace/artifacts/wails-nohost-$(date +%Y%m%d-%H%M%S).png
# or: import -window root /workspace/artifacts/wails-root.png
```

## Mac-only / not expected on this Linux bed

Skip or mark N/A here; verify on macOS:

- Numeric Dock badge / `setBadgeCount` (Wails: Flash + tray tooltip only anyway)
- macOS Dock attention nuances beyond Flash
- `dist:mac` / `dist:mac-smoke` / universal Darwin slices
- macOS notarization / stapling / Sparkle-style update packaging specifics
- Keychain / macOS-only identity helpers
- Any Electron Crashpad/minidump expectations (file-based crash-evidence only)

## Known Linux bed gaps

- No `org.kde.StatusNotifierWatcher` → systray register fails (tray regression limited)
- System D-Bus / login1 sleep-wake blocked (no systemd PID 1, no run/dbus)
- a11y bus missing → set `GTK_A11Y=none` to silence GTK warnings if desired
- Host spawn uses `bash -lc` → **system Node 20 breaks** Host (`findPackageJSON`); keep Node 22 on login PATH (`~/bin/node`)
- Full Host UI may show web authentication gate without credentials (shell+sidecar still valid smoke)
- `wails3 package` AppImage host deps may be incomplete; `go build` binary is the smoke artifact (see docs/wails-package-appimage.md; probe via package-deps:wails)

## Quick re-run script

```bash
export PATH="/home/box/sdk/node-v22.19.0-linux-x64/bin:/home/box/bin:/home/box/sdk/go1.27.0/bin:/home/box/go/bin:$PATH"
export GOPATH=/home/box/go DISPLAY=:8
cd /workspace/repos/dsh-desktop
yarn smoke:wails
yarn build:wails
# window-only
timeout 15s dsh-plugin-desktop/wails/bin/dsh-wails-shell -no-host &
sleep 5; scrot -z /workspace/artifacts/wails-nohost.png
# hybrid (needs Node 22 via bash -lc)
timeout 45s yarn start:wails
```
