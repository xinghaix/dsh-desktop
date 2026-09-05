# User-installed vs bundled dsh (2026-09-05)

## Question

After the Wails hybrid commit, can Desktop switch from the built-in/bundled dsh to a common user-installed dsh (PATH / system prefix)?

## What bundled dsh means here

Not a single binary. Desktop ships a closed Cordis Host stack:

1. Vendored harness runtime under vendor/dsh-runtime, pinned by resolutions and upstream.json.
2. Packaged CLI entry via desktop-cli resolving the @deepseek-ai/dsh package bin.
3. Desktop Cordis Host (host-main) with desktop plugin patches and sidecar announce.
4. Electron Builder still ships the full Host+runtime in asar by default.
5. Wails AppImage today is shell-only; hybrid needs repo layout or an external Host.

Bare public dsh CLI is not a Desktop Host substitute (missing desktop plugin contract).

## How Host is located today (Wails)

HostSidecar.Start: explicit URL, then URL file / command, then autostart bootstrap.
Bootstrap order now: DSH_BIN, then monorepo layout, then PATH dsh-desktop / dsh-plugin-desktop.
See hostbootstrap.go and docs/wails-node-host-boot.md for launcher env knobs.

## Feasibility

- External Desktop Host for Wails shell: yes (DSH_BIN / PATH / host URL knobs).
- System bare dsh as Host: no without redesign.
- Drop vendoring for user global packages: large design risk.
- Optional CLI entry override later: possible with semver gates.

## Recommendation

Keep bundled runtime and Desktop Host as product default.
Use user-installed Host only as AppImage/hybrid escape hatch.
Do not remove vendor runtime without handshake and supported channels.

