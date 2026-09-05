/** Cordis Host launcher selection for Wails hybrid shell (Node-only). */

import { existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

export type HostLauncherMode = "node"

export const DSH_HOST_LAUNCHER_ENV = "DSH_HOST_LAUNCHER"

export interface ResolveHostLauncherOptions {
  readonly environment?: NodeJS.ProcessEnv
  readonly hostMainPath?: string
}

export interface HostLauncherDecision {
  readonly mode: HostLauncherMode
  readonly reason: string
}

function parseMode(raw: string | undefined): HostLauncherMode | "auto" | undefined {
  if (raw === undefined) return undefined
  const normalized = raw.trim().toLowerCase().replace(/_/g, "-")
  if (normalized === "" || normalized === "auto" || normalized === "node") return normalized === "node" ? "node" : "auto"
  if (
    normalized === "electron-as-node"
    || normalized === "electronasnode"
    || normalized === "run-as-node"
    || normalized === "electron-main"
    || normalized === "electron"
    || normalized === "main"
  ) {
    throw new Error(
      `dsh-plugin-desktop: Host launcher mode "${raw}" removed; Electron runtime/packaging is gone. Use Node host-main (DSH_HOST_LAUNCHER=node or omit).`,
    )
  }
  return undefined
}

/** Resolve Host launcher mode (Node only after Electron removal). */
export function resolveHostLauncherMode(options: ResolveHostLauncherOptions = {}): HostLauncherDecision {
  const environment = options.environment ?? process.env
  const hostMainPath = options.hostMainPath
    ?? fileURLToPath(new URL("./host-main.js", import.meta.url))
  const forced = parseMode(environment[DSH_HOST_LAUNCHER_ENV])
  if (forced === "node") {
    return { mode: "node", reason: `${DSH_HOST_LAUNCHER_ENV}=node` }
  }
  if (!existsSync(hostMainPath)) {
    throw new Error(
      `dsh-plugin-desktop: lib/host-main.js missing at ${hostMainPath}; rebuild dsh-plugin-desktop (Electron main fallback removed).`,
    )
  }
  return {
    mode: "node",
    reason: "stock Node Host (Electron launchers removed)",
  }
}

export interface HostSpawnPlan {
  readonly mode: HostLauncherMode
  readonly execPath: string
  readonly argv: readonly string[]
  readonly env: NodeJS.ProcessEnv
  readonly reason: string
}

/** Build spawn plan for Host sidecar (Node host-main only). */
export async function planHostSidecarSpawn(input: {
  readonly hostMainPath: string
  readonly extraArgs?: readonly string[]
  readonly environment?: NodeJS.ProcessEnv
  readonly decision?: HostLauncherDecision
}): Promise<HostSpawnPlan> {
  const environment = { ...(input.environment ?? process.env) }
  const extra = [...(input.extraArgs ?? [])]
  if (!extra.includes("--dsh-wails-host-sidecar")) extra.push("--dsh-wails-host-sidecar")
  environment.DSH_WAILS_HOST_SIDECAR = "1"
  // Strip legacy Electron-as-Node env so children never inherit ABI confusion.
  delete environment.ELECTRON_RUN_AS_NODE
  delete environment.DSH_ALLOW_ELECTRON_MAIN
  delete environment.DSH_HOST_ELECTRON_AS_NODE
  const decision = input.decision ?? resolveHostLauncherMode({
    environment,
    hostMainPath: input.hostMainPath,
  })
  return {
    mode: "node",
    execPath: process.execPath,
    argv: [input.hostMainPath, ...extra],
    env: environment,
    reason: decision.reason,
  }
}
