/** Resolve Desktop data paths without importing Electron. */

import { homedir } from 'node:os'
import { posix, win32 } from 'node:path'
import { DESKTOP_PRODUCT_NAME } from './product-identity.ts'

/**
 * Electron-compatible userData directory for DSH Desktop.
 * Mirrors `defaultDesktopUserDataDirectory` in bin.ts (keep in sync).
 */
export function resolveDesktopUserDataDirectory(
  platform: NodeJS.Platform = process.platform,
  environment: NodeJS.ProcessEnv = process.env,
  homeDirectory: string = homedir(),
): string {
  const path = platform === 'win32' ? win32 : posix
  if (platform === 'win32') {
    const appData = environment.APPDATA
    if (appData === undefined || appData.length === 0) {
      throw new Error('APPDATA is unavailable; cannot locate DSH Desktop userData')
    }
    return path.join(appData, DESKTOP_PRODUCT_NAME)
  }
  if (platform === 'darwin') {
    return path.join(homeDirectory, 'Library', 'Application Support', DESKTOP_PRODUCT_NAME)
  }
  const config = environment.XDG_CONFIG_HOME
  return path.join(
    config === undefined || config.length === 0 ? path.join(homeDirectory, '.config') : config,
    DESKTOP_PRODUCT_NAME,
  )
}

/** Electron-compatible appData parent (Roaming / Application Support / .config). */
export function resolveDesktopAppDataDirectory(
  platform: NodeJS.Platform = process.platform,
  environment: NodeJS.ProcessEnv = process.env,
  homeDirectory: string = homedir(),
): string {
  const path = platform === 'win32' ? win32 : posix
  if (platform === 'win32') {
    const appData = environment.APPDATA
    if (appData === undefined || appData.length === 0) {
      throw new Error('APPDATA is unavailable; cannot locate Desktop appData')
    }
    return appData
  }
  if (platform === 'darwin') {
    return path.join(homeDirectory, 'Library', 'Application Support')
  }
  const config = environment.XDG_CONFIG_HOME
  return config === undefined || config.length === 0
    ? path.join(homeDirectory, '.config')
    : config
}

/** Home directory used for shell-environment recovery (Electron app.getPath('home')). */
export function resolveDesktopHomeDirectory(
  environment: NodeJS.ProcessEnv = process.env,
  homeDirectory: string = homedir(),
): string {
  const override = environment.HOME ?? environment.USERPROFILE
  if (override !== undefined && override.length > 0) return override
  return homeDirectory
}
