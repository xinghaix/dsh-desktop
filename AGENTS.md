# DSH Desktop repository rules

This repository owns the desktop product around an unmodified DeepSeek Harness checkout.

## Prerequisites and setup

- Use Node.js `^22.19.0` or `>=24.0.0` and the root Yarn `4.18.0` release through Corepack.
- Initialize the pinned upstream checkout with `git submodule update --init --recursive`.
- Install root dependencies with `corepack yarn install --immutable`.

## Build, run, and verify

- Primary product path: `corepack yarn start:wails` / `dev:wails` / `smoke:wails`, plus `corepack yarn start:host` for the Node Cordis Host sidecar. Packaging: `package:wails`. See `docs/wails-migration.md` and `docs/wails-workspace-scripts.md`.
- Build the desktop package with `corepack yarn build`. Prefer `package:wails` for native shell artifacts.
- Run unit tests with `corepack yarn test`.
- Run type checking with `corepack yarn typecheck`.
- Run the complete headless gate with `corepack yarn check`.
- Run upstream operations through the root scripts, such as `corepack yarn upstream:build`.

- `deepseek-harness/` is a pinned upstream Git submodule. Never edit files inside it from a desktop feature branch.
- `dsh-plugin-desktop/` owns the Cordis Host and Client faces, the Wails native shell under `wails/`, Node Host bootstrap, packaging, and release tests. Product packaging is `package:wails` (CI flip still outstanding — see debt doc).
- `dsh-plugin-desktop/wails/` owns the primary Go 1.27 + Wails v3 native shell with a Node Cordis Host sidecar. Do not edit `deepseek-harness/` from Wails work. Canonical maps: `docs/wails-migration.md`, `dsh-plugin-desktop/src/wails-shell-bridge.md`, `docs/wails-node-host-boot.md`.
- `dsh-community-fabric/` owns the community interoperability RFC. Until schemas and a reviewed reference adapter exist, it remains a private documentation scaffold and must not declare loadable DSH or package entry points.
- `dsh-community-market/` owns the community-market shell. Until its runtime is implemented, it remains a private documentation scaffold and must not declare loadable DSH or package entry points.
- The outer repository and all owned packages use the root Yarn release with `nodeLinker: node-modules`.
- The upstream submodule keeps its own pnpm workspace. Run upstream commands through the root `upstream:*` scripts, whose Yarn portable-shell commands enter the submodule before invoking Corepack.
- Compatibility mode must run the upstream default client without overrides. Advanced presentation belongs to desktop-owned client plugins and may replace documented slots or services through profile composition.
- Keep graphical application launch explicit. Builds, typechecks, unit tests, and Loader smokes must remain headless-safe.
- Commit before major changes of direction and keep the submodule pin update separate from desktop behavior changes.
- Keep the repository topology and package-manager split consistent with the [owning Agent Note](.agents/notes/implemented/process/2026-08-15-pinned-upstream-and-isolated-yarn-workspace.md).
