/** Hybrid Wails shell: Cordis Host boots without Electron BrowserWindow/Tray. */

import { writeFileSync } from 'node:fs'
import { DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT } from './relaunch-arguments.ts'

export { DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT }

/** Process env that selects Host-only boot for the Wails native shell. */
export const DSH_WAILS_HOST_SIDECAR_ENV = 'DSH_WAILS_HOST_SIDECAR'

/** Stdout line prefix consumed by the Go HostSidecar waiter. */
export const DSH_HOST_READY_PREFIX = 'DSH_HOST_READY '

/**
 * Detect Wails Host-sidecar mode from env or the exact argv marker.
 * @param argv - full process argv (defaults to `process.argv`).
 * @param environment - process env (defaults to `process.env`).
 */
export function desktopWailsHostSidecarRequested(
  argv: readonly string[] = process.argv,
  environment: NodeJS.ProcessEnv = process.env,
): boolean {
  if (environment[DSH_WAILS_HOST_SIDECAR_ENV] === '1') return true
  return argv.slice(1).includes(DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT)
}

/**
 * Announce the Host UI URL for the Wails shell (stdout + optional URL file).
 * Matches `wails/scripts/announce-host-ready.mjs`.
 * @param url - authenticated loopback UI URL the Wails webview should load.
 * @param environment - process env that may set `DSH_HOST_URL_FILE`.
 */
export function announceWailsHostReady(
  url: string,
  environment: NodeJS.ProcessEnv = process.env,
): void {
  const trimmed = url.trim()
  if (trimmed.length === 0) {
    throw new Error('dsh-plugin-desktop: Wails Host ready URL must not be empty')
  }
  process.stdout.write(`${DSH_HOST_READY_PREFIX}${trimmed}\n`)
  const file = environment.DSH_HOST_URL_FILE
  if (file !== undefined && file.trim().length > 0) {
    writeFileSync(file.trim(), `${trimmed}\n`, 'utf8')
  }
}
