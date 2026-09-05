# Wails workspace scripts

Recommended hybrid entry (repo root or dsh-plugin-desktop):

- start:wails / dev:wails / build:wails / package:wails / smoke:wails
- start:host / start:host:node
- sync:native-ui:wails (copies lib/native-ui into wails embed tree)
- node scripts/wails-smoke.mjs (non-workflow smoke)
- node dsh-plugin-desktop/wails/scripts/run-wails.mjs package-deps (AppImage tooling probe)

Helper: dsh-plugin-desktop/wails/scripts/run-wails.mjs

Go resolution: DSH_GO_BIN -> PATH go -> ~/sdk/go1.27.0/bin -> ~/go/bin -> /home/box/sdk/go1.27.0/bin

## Packaging

- package:wails prefers wails3 package; falls back to bin/dsh-wails-shell.
- AppImage host deps and legacy-builder primacy: docs/wails-package-appimage.md.
- smoke:wails always runs go test + go build, then prints the package-deps probe.

## Enabling CI smoke when workflow scope exists

Product path is Wails+Node Host. Packaging CI still needs flip off removed scripts; wails-smoke example stays until workflow scope.

While the push token lacks workflow scope:

1. Keep iterating on docs/wails-ci-smoke.yml.example only (already in-tree).
2. Run smoke:wails / node scripts/wails-smoke.mjs locally or on the cloud box.

When a credential with workflow scope is available:

```bash
cp docs/wails-ci-smoke.yml.example .github/workflows/wails-smoke.yml
git add .github/workflows/wails-smoke.yml
git commit -m "ci(wails): add additive wails-smoke workflow"
git push origin feat/wails-user-dsh-home   # xinghaix remote in this fork
```

Do not resurrect retired packaging scripts. Prefer package:wails / smoke:wails; see docs/wails-migration.md and the debt doc.


## Credential note (xinghaix fork)

`gh auth status` (2026-09-05): scopes gist, read:org, repo — **no workflow**. Live wails-smoke
workflow stays blocked; keep iterating the example + local smoke.
