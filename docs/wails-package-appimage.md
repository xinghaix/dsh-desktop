# package:wails and Linux AppImage dependencies

## Product packaging vs hybrid smoke

| Path | Role |
| --- | --- |
| electron-builder (dist scripts / product CI workflows) | Default product CI and release artifacts until the explicit release flip |
| package:wails | Parallel Wails packaging path (wails3 package when available) |
| smoke:wails / scripts/wails-smoke.mjs | Go test + go build of bin/dsh-wails-shell (always the hybrid smoke artifact) |

Do not treat a successful package:wails fallback Go binary as a shipped Linux AppImage.

## What package:wails does

Entry: root / workspace package:wails -> dsh-plugin-desktop/wails/scripts/run-wails.mjs package.

1. If wails3 is on PATH, run wails3 package (Taskfile Linux targets may build AppImage/deb/rpm/aur).
2. If wails3 is missing or packaging fails on this host, fall back to go build -o bin/dsh-wails-shell with a clear stderr note.
3. AppImage creation (when wails3 packaging works) uses wails/build/linux/appimage/ (linuxdeploy AppImage fetch + FUSE).

## Host dependencies for a real AppImage

Minimum for the installer path (not required for smoke:wails):

- Go 1.27+ (same as shell build)
- wails3 CLI matching the module (beta.16 on this branch)
- Network to fetch linuxdeploy AppImages (see build/linux/appimage/build.sh)
- FUSE / fusermount so linuxdeploy AppImages can run on the packager host
- GTK / WebKitGTK build libs already needed to compile the shell
- file(1) / libmagic — required by appimagetool (blocked AppImage until apt install file on 2026-09-05 bed)
- Optional: wget (used by the vendor AppImage script)

Informational probe (does not fail smoke):

```bash
node dsh-plugin-desktop/wails/scripts/run-wails.mjs package-deps
# smoke:wails ends with the same probe after go test/build
```

On this cloud box, expect wails3 and/or FUSE to be MISS; use the Go binary for hybrid verification.

## Release flip gate

AppImage/package:wails becoming the *shipping* Linux channel requires the release flip
checklist in docs/wails-migration.md (signed artifacts, AuthProxy on package, CI green).
Until then package:wails is R&D / preview only.

## CI note

The live workflow file under .github/workflows is not committed while the push credential lacks workflow scope.
Keep docs/wails-ci-smoke.yml.example in sync and copy it when scope exists (see docs/wails-workspace-scripts.md).
Product release CI stays electron-builder until flipped.


Token scopes observed on xinghaix cloud credential (2026-09-05): gist, read:org, repo — missing workflow.
Do not commit .github/workflows/wails-smoke.yml until workflow scope exists.

## Bed evidence (2026-09-05)

After installing `file`, `node wails/scripts/run-wails.mjs package` produced (gitignored under wails/bin/):

- dsh-wails-shell-x86_64.AppImage (~102MiB)
- dsh-wails-shell.deb / .rpm / .pkg.tar.zst
- bin/dsh-wails-shell native binary

Still **not** the product download channel (electron-builder remains default until flip).
