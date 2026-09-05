/** Cordis Host launcher selection for Wails hybrid shell. */

import { existsSync, readdirSync } from 'node:fs'
import { homedir } from 'node:os'
import { join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

export type HostLauncherMode = "node" | "electron-as-node" | "electron-main"

export const DSH_HOST_ELECTRON_AS_NODE_ENV = "DSH_HOST_ELECTRON_AS_NODE"
export const DSH_HOST_LAUNCHER_ENV = "DSH_HOST_LAUNCHER"
/** Opt-in for LAST-RESORT Electron main Host (hybrid prefers Node / Electron-as-Node). */
export const DSH_ALLOW_ELECTRON_MAIN_ENV = "DSH_ALLOW_ELECTRON_MAIN"

const TRUTHY = new Set(["1", "true", "yes", "on"])
const FALSY = new Set(["0", "false", "no", "off"])

export interface ResolveHostLauncherOptions {
  readonly environment?: NodeJS.ProcessEnv
  readonly userDataDir?: string
  readonly scanRoots?: readonly string[]
  readonly hostMainPath?: string
  readonly electronPath?: string | undefined
  readonly maxProbeEntries?: number
  readonly resolveUserDataDir?: () => string
}

export interface HostLauncherDecision {
  readonly mode: HostLauncherMode
  readonly reason: string
  readonly electronNativeAddonsLikely: boolean
}

function envFlag(value: string | undefined): boolean | undefined {
  if (value === undefined) return undefined
  const normalized = value.trim().toLowerCase()
  if (TRUTHY.has(normalized)) return true
  if (FALSY.has(normalized)) return false
  return undefined
}

function parseMode(raw: string | undefined): HostLauncherMode | "auto" | undefined {
  if (raw === undefined) return undefined
  const normalized = raw.trim().toLowerCase().replace(/_/g, "-")
  if (normalized === "" || normalized === "auto") return "auto"
  if (normalized === "node") return "node"
  if (normalized === "electron-as-node" || normalized === "electronasnode" || normalized === "run-as-node") {
    return "electron-as-node"
  }
  if (normalized === "electron-main" || normalized === "electron" || normalized === "main") {
    return "electron-main"
  }
  return undefined
}

function defaultUserDataDir(environment: NodeJS.ProcessEnv): string {
  const platform = process.platform
  const home = environment.HOME ?? environment.USERPROFILE ?? homedir()
  if (platform === "win32") {
    const appData = environment.APPDATA
    if (appData === undefined || appData.length === 0) {
      return join(home, "AppData", "Roaming", "DSH Desktop")
    }
    return join(appData, "DSH Desktop")
  }
  if (platform === "darwin") return join(home, "Library", "Application Support", "DSH Desktop")
  const config = environment.XDG_CONFIG_HOME
  return join(config === undefined || config.length === 0 ? join(home, ".config") : config, "DSH Desktop")
}

/** Profiles are materialized with npm_config_runtime=electron; node_modules => Electron ABI likely. */
export function electronNativeAddonsLikelyRequired(
  roots: readonly string[],
): boolean {
  for (const root of roots) {
    if (!existsSync(root)) continue
    let entries: string[]
    try { entries = readdirSync(root) } catch { continue }
    for (const name of entries) {
      if (existsSync(join(root, name, "node_modules"))) return true
      if (name === "node_modules" && existsSync(join(root, name))) return true
    }
    if (existsSync(join(root, "node_modules"))) return true
  }
  return false
}

export function defaultNativeAddonScanRoots(
  userDataDir: string,
  pluginDir: string = fileURLToPath(new URL("..", import.meta.url)),
): string[] {
  return [
    join(userDataDir, "profiles"),
    join(userDataDir, "Safe Mode"),
    pluginDir,
  ]
}

/** Resolve Host launcher mode (env overrides, then profile node_modules heuristic). */
export function resolveHostLauncherMode(options: ResolveHostLauncherOptions = {}): HostLauncherDecision {
  const environment = options.environment ?? process.env
  const userDataDir = options.userDataDir
    ?? (options.resolveUserDataDir ? options.resolveUserDataDir() : defaultUserDataDir(environment))
  const hostMainPath = options.hostMainPath
    ?? fileURLToPath(new URL("./host-main.js", import.meta.url))
  const hostMainExists = existsSync(hostMainPath)
  const scanRoots = options.scanRoots ?? defaultNativeAddonScanRoots(userDataDir)
  const electronNativeAddonsLikely = electronNativeAddonsLikelyRequired(scanRoots)
  const forced = parseMode(environment[DSH_HOST_LAUNCHER_ENV])
  if (forced === "node" || forced === "electron-as-node" || forced === "electron-main") {
    return { mode: forced, reason: `${DSH_HOST_LAUNCHER_ENV}=${forced}`, electronNativeAddonsLikely }
  }
  const asNodeFlag = envFlag(environment[DSH_HOST_ELECTRON_AS_NODE_ENV])
  if (asNodeFlag === true) {
    return { mode: "electron-as-node", reason: `${DSH_HOST_ELECTRON_AS_NODE_ENV}=1`, electronNativeAddonsLikely }
  }
  if (asNodeFlag === false && hostMainExists) {
    return { mode: "node", reason: `${DSH_HOST_ELECTRON_AS_NODE_ENV}=0`, electronNativeAddonsLikely }
  }
  if (!hostMainExists) {
    return { mode: "electron-main", reason: "lib/host-main.js missing", electronNativeAddonsLikely }
  }
  if (electronNativeAddonsLikely) {
    return {
      mode: "electron-as-node",
      reason: "profile node_modules present; prefer ELECTRON_RUN_AS_NODE for Electron ABI natives",
      electronNativeAddonsLikely,
    }
  }
  return {
    mode: "node",
    reason: "no profile node_modules detected; stock Node Host",
    electronNativeAddonsLikely,
  }
}

/** Resolve the Electron executable path from the electron package or ELECTRON_PATH. */
export async function resolveElectronExecutable(
  environment: NodeJS.ProcessEnv = process.env,
): Promise<string | undefined> {
  const fromEnv = environment.ELECTRON_PATH?.trim()
  if (fromEnv && existsSync(fromEnv)) return resolve(fromEnv)
  try {
    const imported = await import("electron") as { default?: unknown }
    if (typeof imported.default === "string" && existsSync(imported.default)) {
      return imported.default
    }
  } catch {
    // peer may be absent in pure Node installs
  }
  return undefined
}

export interface HostSpawnPlan {
  readonly mode: HostLauncherMode
  readonly execPath: string
  readonly argv: readonly string[]
  readonly env: NodeJS.ProcessEnv
  readonly reason: string
}

/** Build spawn plan for Host sidecar. */
export async function planHostSidecarSpawn(input: {
  readonly hostMainPath: string
  readonly electronMainPath: string
  readonly extraArgs?: readonly string[]
  readonly environment?: NodeJS.ProcessEnv
  readonly decision?: HostLauncherDecision
  readonly scanRoots?: readonly string[]
}): Promise<HostSpawnPlan> {
  const environment = { ...(input.environment ?? process.env) }
  const extra = [...(input.extraArgs ?? [])]
  if (!extra.includes("--dsh-wails-host-sidecar")) extra.push("--dsh-wails-host-sidecar")
  environment.DSH_WAILS_HOST_SIDECAR = "1"
  const electronPath = await resolveElectronExecutable(environment)
  const decision = input.decision ?? resolveHostLauncherMode({
    environment,
    hostMainPath: input.hostMainPath,
    electronPath,
    ...(input.scanRoots === undefined ? {} : { scanRoots: input.scanRoots }),
  })

  if (decision.mode === "electron-main") {
    const allow = envFlag(environment[DSH_ALLOW_ELECTRON_MAIN_ENV])
    const forced = parseMode(environment[DSH_HOST_LAUNCHER_ENV]) === "electron-main"
    // Hybrid product path refuses Electron main unless explicitly opted in.
    if (!forced && allow !== true) {
      throw new Error(
        "dsh-plugin-desktop: Electron main Host blocked (set DSH_ALLOW_ELECTRON_MAIN=1 or DSH_HOST_LAUNCHER=electron-main). Prefer host-main Node / Electron-as-Node.",
      )
    }
    if (!electronPath) {
      throw new Error("dsh-plugin-desktop: Electron executable required for electron-main Host launcher")
    }
    return {
      mode: "electron-main",
      execPath: electronPath,
      argv: [input.electronMainPath, ...extra],
      env: environment,
      reason: `${decision.reason} (LAST-RESORT Electron main; hybrid prefers Node host-main)`,
    }
  }

  if (decision.mode === "electron-as-node") {
    if (!electronPath) {
      return {
        mode: "node",
        execPath: process.execPath,
        argv: [input.hostMainPath, ...extra],
        env: environment,
        reason: `${decision.reason}; Electron binary missing — using stock Node`,
      }
    }
    return {
      mode: "electron-as-node",
      execPath: electronPath,
      argv: [input.hostMainPath, ...extra],
      env: { ...environment, ELECTRON_RUN_AS_NODE: "1" },
      reason: decision.reason,
    }
  }

  return {
    mode: "node",
    execPath: process.execPath,
    argv: [input.hostMainPath, ...extra],
    env: environment,
    reason: decision.reason,
  }
}
