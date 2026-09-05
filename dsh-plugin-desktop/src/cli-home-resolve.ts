/** Home-first resolution for the public `dsh` CLI entry (`@deepseek-ai/dsh`). */

import { existsSync, readFileSync } from 'node:fs'
import { homedir } from 'node:os'
import { join, resolve } from 'node:path'

/** Env: absolute path to `@deepseek-ai/dsh` `lib/bin.js` (or compatible CLI entry). */
export const DSH_CLI_BIN_ENV = 'DSH_CLI_BIN'
/** Env: harness / install home override (same root as Cordis `DSH_HOME`). */
export const DSH_HOME_ENV = 'DSH_HOME'
/**
 * Opt-in: allow falling back to the Desktop-packaged `@deepseek-ai/dsh` tree.
 * Off by default so terminal / desktop-cli never silently uses the ASAR copy.
 */
export const DSH_CLI_ALLOW_BUNDLED_ENV = 'DSH_CLI_ALLOW_BUNDLED'

const CLI_RELATIVE_ENTRIES = [
  join('node_modules', '@deepseek-ai', 'dsh', 'lib', 'bin.js'),
  join('lib', 'bin.js'),
] as const

export interface ResolveDesktopCliEntryOptions {
  readonly environment?: NodeJS.ProcessEnv
  readonly homeDirectory?: string
  /** Packaged ASAR / node_modules entry used only when `DSH_CLI_ALLOW_BUNDLED` is set. */
  readonly bundledEntryPath?: string
}

export interface DesktopCliEntryHit {
  readonly path: string
  readonly reason: string
}

export interface DesktopCliEntryReport {
  readonly checked: readonly string[]
  readonly hit?: DesktopCliEntryHit
  /** Bilingual message when missing (or short confirm when hit). */
  readonly message: string
}

function envTruthy(value: string | undefined): boolean {
  if (value === undefined) return false
  switch (value.trim().toLowerCase()) {
    case '1':
    case 'true':
    case 'yes':
    case 'on':
      return true
    default:
      return false
  }
}

function expandHome(path: string, homeDirectory: string): string {
  if (path === '~') return homeDirectory
  if (path.startsWith('~/') || path.startsWith('~\\')) return join(homeDirectory, path.slice(2))
  return path
}

function appendUnique(dst: string[], value: string): void {
  if (value.length === 0 || dst.includes(value)) return
  dst.push(value)
}

/** Candidate install roots for a user `dsh` CLI (not Desktop Host). */
export function candidateCliHomeRoots(
  environment: NodeJS.ProcessEnv = process.env,
  homeDirectory: string = homedir(),
): string[] {
  const roots: string[] = []
  const fromEnv = environment[DSH_HOME_ENV]
  if (fromEnv !== undefined && fromEnv.trim().length > 0) {
    appendUnique(roots, resolve(expandHome(fromEnv.trim(), homeDirectory)))
  }
  appendUnique(roots, resolve(join(homeDirectory, '.dsh')))
  appendUnique(roots, resolve(join(homeDirectory, 'dsh')))
  const xdg = environment.XDG_DATA_HOME
  if (xdg !== undefined && xdg.trim().length > 0) {
    appendUnique(roots, resolve(join(xdg.trim(), 'dsh')))
  } else {
    appendUnique(roots, resolve(join(homeDirectory, '.local', 'share', 'dsh')))
  }
  return roots
}

function packageNameAt(root: string): string | undefined {
  const manifest = join(root, 'package.json')
  if (!existsSync(manifest)) return undefined
  try {
    const raw: unknown = JSON.parse(readFileSync(manifest, 'utf8'))
    if (raw !== null && typeof raw === 'object') {
      const name = (raw as { name?: unknown }).name
      return typeof name === 'string' ? name : undefined
    }
  } catch {
    return undefined
  }
  return undefined
}

function probeRoot(root: string, label: string, checked: string[]): DesktopCliEntryHit | undefined {
  for (const rel of CLI_RELATIVE_ENTRIES) {
    const candidate = join(root, rel)
    appendUnique(checked, candidate)
    if (!existsSync(candidate)) continue
    // Desktop Host homes also expose lib/bin.js (dsh-desktop launcher). Only accept bare
    // lib/bin.js when package.json names the public CLI.
    if (rel === join('lib', 'bin.js') && packageNameAt(root) !== '@deepseek-ai/dsh') {
      continue
    }
    return { path: candidate, reason: `${label}:${rel}` }
  }
  return undefined
}

function friendlyMissingMessage(checked: readonly string[]): string {
  const listed = checked.slice(0, 24).map(function (p) { return "  - " + p }).join("\n")
  const more = checked.length > 24 ? ("\n  ... and " + String(checked.length - 24) + " more") : ""
  const lines = [
    "No user-installed dsh CLI (@deepseek-ai/dsh) was found.",
    "Desktop will not silently use the bundled package copy.",
    "",
    "Checked paths:",
    listed.length === 0 ? "  (none)" : listed + more,
    "",
    "Next steps:",
    "  1. Install @deepseek-ai/dsh under the user home directory",
    "  2. Or point the CLI bin / home override variables",
    "  3. Or enable the documented bundled opt-in flag",
    "",
    "Note: Host is Cordis Web UI (dsh web profile family); Wails may launch a packaged Host. This rule only forces CLI entry to use the user home dsh.",
  ]
  return lines.join("\n")
}

/** Resolve public dsh CLI entry home-first. */
export function resolveDesktopCliEntry(options: ResolveDesktopCliEntryOptions = {}): DesktopCliEntryReport {
  const environment = options.environment ?? process.env
  const homeDirectory = options.homeDirectory ?? (environment.HOME && environment.HOME.length > 0 ? environment.HOME : (environment.USERPROFILE && environment.USERPROFILE.length > 0 ? environment.USERPROFILE : homedir()))
  const checked: string[] = []

  const cliBin = environment[DSH_CLI_BIN_ENV]
  if (cliBin !== undefined && cliBin.trim().length > 0) {
    const path = resolve(expandHome(cliBin.trim(), homeDirectory))
    appendUnique(checked, DSH_CLI_BIN_ENV + "=" + path)
    if (existsSync(path)) {
      return {
        checked,
        hit: { path, reason: DSH_CLI_BIN_ENV },
        message: "dsh CLI via " + DSH_CLI_BIN_ENV + " -> " + path,
      }
    }
  } else {
    appendUnique(checked, DSH_CLI_BIN_ENV + "=(unset)")
  }

  const dshHomeRaw = environment[DSH_HOME_ENV]
  if (dshHomeRaw === undefined || dshHomeRaw.trim().length === 0) {
    appendUnique(checked, DSH_HOME_ENV + "=(unset)")
  }

  for (const root of candidateCliHomeRoots(environment, homeDirectory)) {
    const fromEnv = dshHomeRaw !== undefined && dshHomeRaw.trim().length > 0
      && resolve(expandHome(dshHomeRaw.trim(), homeDirectory)) === root
    let label: string
    if (fromEnv) {
      label = DSH_HOME_ENV
    } else if (root.startsWith(homeDirectory)) {
      const rel = root.slice(homeDirectory.length).replace(/^[/\\]/u, "")
      label = "HOME:~/" + rel.split(/[/\\]/u).join("/")
    } else {
      label = "HOME:" + root
    }
    const hit = probeRoot(root, label, checked)
    if (hit !== undefined) {
      return {
        checked,
        hit,
        message: "dsh CLI via " + hit.reason + " -> " + hit.path,
      }
    }
  }

  if (envTruthy(environment[DSH_CLI_ALLOW_BUNDLED_ENV]) && options.bundledEntryPath !== undefined) {
    appendUnique(checked, "bundled:" + options.bundledEntryPath)
    if (existsSync(options.bundledEntryPath)) {
      return {
        checked,
        hit: { path: options.bundledEntryPath, reason: DSH_CLI_ALLOW_BUNDLED_ENV },
        message: "dsh CLI via opt-in bundled package -> " + options.bundledEntryPath,
      }
    }
  } else {
    appendUnique(checked, DSH_CLI_ALLOW_BUNDLED_ENV + "=(off)")
  }

  return {
    checked,
    message: friendlyMissingMessage(checked),
  }
}

/** Throw a friendly Error when the CLI entry cannot be resolved. */
export function requireDesktopCliEntry(options: ResolveDesktopCliEntryOptions = {}): DesktopCliEntryHit {
  const report = resolveDesktopCliEntry(options)
  if (report.hit === undefined) {
    throw new Error(report.message)
  }
  return report.hit
}
