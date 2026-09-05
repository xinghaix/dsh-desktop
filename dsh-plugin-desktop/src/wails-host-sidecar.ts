/** Hybrid Wails shell: Cordis Host boots without Electron BrowserWindow/Tray. */

import { writeFileSync } from 'node:fs'
import { DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT } from './relaunch-arguments.ts'

export { DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT }

/** Process env that selects Host-only boot for the Wails native shell. */
export const DSH_WAILS_HOST_SIDECAR_ENV = 'DSH_WAILS_HOST_SIDECAR'

/** Stdout line prefix consumed by the Go HostSidecar waiter. */
export const DSH_HOST_READY_PREFIX = 'DSH_HOST_READY '

/** Stdout line prefix for the Desktop renderer access header (Wails BridgeService). */
export const DSH_HOST_AUTH_HEADER_PREFIX = 'DSH_HOST_AUTH_HEADER '

/** Stdout line when Host needs Wails-owned recovery UI instead of Electron. */
export const DSH_HOST_RECOVERY_REQUIRED_PREFIX = 'DSH_HOST_RECOVERY_REQUIRED '

/** Stdout line when Host advertises LAN HTTPS edge state for Wails. */
export const DSH_HOST_LAN_HTTPS_PREFIX = 'DSH_HOST_LAN_HTTPS '

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
 * Electron-light Host: skip BrowserWindow / Tray / local dialog construction.
 * Same detector as sidecar today; kept as a named alias for call sites that
 * care about GUI omission rather than announce protocol.
 */
export function desktopWailsSkipElectronGui(
  argv: readonly string[] = process.argv,
  environment: NodeJS.ProcessEnv = process.env,
): boolean {
  return desktopWailsHostSidecarRequested(argv, environment)
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

/**
 * Announce the renderer access header for the Wails auth bridge.
 * WebKitGTK cannot inject per-request headers; the Wails shell prefers a
 * loopback auth proxy that injects this header. Sidecar mode may still enable
 * ordinary-browser loopback access as a fallback when the proxy is unavailable.
 */
export function announceWailsHostAuthHeader(
  name: string,
  value: string,
  environment: NodeJS.ProcessEnv = process.env,
): void {
  const headerName = name.trim()
  const headerValue = value.trim()
  if (headerName.length === 0 || headerValue.length === 0) {
    throw new Error('dsh-plugin-desktop: Wails Host auth header requires name and value')
  }
  process.stdout.write(`${DSH_HOST_AUTH_HEADER_PREFIX}${headerName} ${headerValue}\n`)
  const file = environment.DSH_HOST_AUTH_META_FILE
  if (file !== undefined && file.trim().length > 0) {
    writeFileSync(file.trim(), `header=${headerName} ${headerValue}\n`, 'utf8')
  }
}

/**
 * Ask the Wails shell to open recovery UI; Host stays Electron-light (no BrowserWindow).
 */
export function announceWailsHostRecoveryRequired(
  detail: string,
  environment: NodeJS.ProcessEnv = process.env,
): void {
  const message = detail.trim() || 'recovery-required'
  process.stdout.write(`${DSH_HOST_RECOVERY_REQUIRED_PREFIX}${message}\n`)
  const file = environment.DSH_HOST_RECOVERY_FILE
  if (file !== undefined && file.trim().length > 0) {
    writeFileSync(file.trim(), `${message}\n`, 'utf8')
  }
}

/**
 * Announce LAN HTTPS edge snapshot fields for the Wails CapabilitiesService.
 * Format: `DSH_HOST_LAN_HTTPS state=<s> port=<n|null> addresses=<csv> fingerprint=<fp|null> error=<code|null>`
 */
export function announceWailsHostLanHttps(
  snapshot: {
    readonly state: string
    readonly actualPort: number | null
    readonly addresses: readonly string[]
    readonly caFingerprint: string | null
    readonly errorCode: string | null
    readonly urls?: readonly string[]
  },
): void {
  const addresses = snapshot.addresses.length === 0 ? '-' : snapshot.addresses.join(',')
  const port = snapshot.actualPort === null ? 'null' : String(snapshot.actualPort)
  const fingerprint = snapshot.caFingerprint ?? 'null'
  const error = snapshot.errorCode ?? 'null'
  const urls = snapshot.urls === undefined || snapshot.urls.length === 0
    ? '-'
    : snapshot.urls.join(',')
  process.stdout.write(
    `${DSH_HOST_LAN_HTTPS_PREFIX}state=${snapshot.state} port=${port} addresses=${addresses} fingerprint=${fingerprint} error=${error} urls=${urls}\n`,
  )
}
