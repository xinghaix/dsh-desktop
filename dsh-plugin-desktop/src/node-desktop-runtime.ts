/** Node / Wails Host-sidecar DesktopRuntime: no Electron BrowserWindow or Tray. */

import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import type { DesktopLogger } from './desktop-logger.ts'
import type { DesktopInstallationId } from './desktop-installation-id.ts'
import { DESKTOP_RELEASE_CHANNEL } from './product-identity.ts'
import type { RendererBootReport } from './renderer-boot-contract.ts'
import type {
  DesktopLocale,
  DesktopNotification,
  DesktopPlatform,
  DesktopRuntime,
  DesktopShellSpec,
  DesktopThemeSource,
  DesktopTrayItem,
  DesktopTrayItemRegistration,
  DesktopUpdateAdapter,
} from './runtime.ts'
import { desktopLocaleFromLanguageTag } from './tray-locale.ts'
import type { ProfileCreateWindowOptions } from './profile-create-window.ts'
import type { UpdateCheckResult } from './update-checker.ts'
import { windowsBuildNumber } from './window-material.ts'

function nodeHostProductVersion(moduleUrl: string = import.meta.url): string {
  const value: unknown = JSON.parse(readFileSync(new URL('../package.json', moduleUrl), 'utf8'))
  if (value === null || typeof value !== 'object' || typeof (value as { version?: unknown }).version !== 'string') {
    throw new Error('dsh-plugin-desktop: package.json has no product version')
  }
  return (value as { version: string }).version
}

const PRODUCT_VERSION = nodeHostProductVersion()

function nodePlatform(): DesktopPlatform {
  if (process.platform === 'darwin' || process.platform === 'win32' || process.platform === 'linux') {
    return process.platform
  }
  throw new Error(`dsh-plugin-desktop: unsupported Node Host platform ${process.platform}`)
}

function languageTagFromEnv(environment: NodeJS.ProcessEnv = process.env): string {
  const raw = environment.LC_ALL ?? environment.LC_MESSAGES ?? environment.LANG ?? 'en'
  return raw.split('.')[0]?.replace('_', '-') || 'en'
}

/**
 * Headless DesktopRuntime for Cordis Host under Wails (or plain Node).
 * Host plugins may call schedule/tray/updates; native shell surfaces are deferred to Wails.
 */
export class NodeDesktopRuntime implements DesktopRuntime {
  readonly platform: DesktopPlatform
  readonly windowsBuild: number | undefined
  readonly updates: DesktopUpdateAdapter

  private currentLocale: DesktopLocale
  private scheduled: DesktopShellSpec | undefined
  private quitting = false
  private readonly trayItems = new Map<symbol, DesktopTrayItem>()
  private restartRequest: Promise<void> | undefined

  constructor(
    private readonly restart: (target?: 'recovery') => Promise<void>,
    private readonly logger: DesktopLogger | undefined = undefined,
    private readonly userDataDir: string,
    installationId?: DesktopInstallationId,
    initialLocale: DesktopLocale = desktopLocaleFromLanguageTag(languageTagFromEnv()),
  ) {
    this.platform = nodePlatform()
    this.windowsBuild = this.platform === 'win32' ? windowsBuildNumber() : undefined
    this.currentLocale = initialLocale
    const statePath = join(userDataDir, 'updates', 'state.json')
    this.updates = {
      isPackaged: false,
      canDownload: false,
      currentVersion: PRODUCT_VERSION,
      releaseChannel: DESKTOP_RELEASE_CHANNEL,
      statePath,
      ...(installationId === undefined ? {} : { installationId }),
      request: (url, init) => fetch(url, init),
      confirmDownload: async () => false,
      showManualCheckResult: async (result: UpdateCheckResult | null) => {
        this.logInfo(`Node Host update check: ${result === null ? 'null' : JSON.stringify(result)}`)
      },
      downloadAndOpen: async () => {
        throw new Error('dsh-plugin-desktop: Node Host does not download Electron installers')
      },
      notify: notification => { this.showNotification(notification) },
    }
  }

  private logInfo(message: string): void {
    if (this.logger !== undefined) this.logger.error(message)
    else process.stderr.write(`${message}\n`)
  }

  private showNotification(notification: DesktopNotification): void {
    this.logInfo(`dsh-plugin-desktop[node-host]: ${notification.title}: ${notification.body}`)
  }

  get locale(): DesktopLocale {
    return this.currentLocale
  }

  schedule(spec: DesktopShellSpec): () => Promise<void> {
    if (this.scheduled !== undefined) {
      throw new Error('dsh-plugin-desktop: a native shell generation is already registered')
    }
    this.scheduled = spec
    let disposed = false
    return async () => {
      if (disposed) return
      disposed = true
      if (this.scheduled === spec) this.scheduled = undefined
    }
  }

  async mountScheduled(_beforeInteractive?: () => void): Promise<void> {
    // Wails owns the webview; Node Host never mounts Electron BrowserWindow.
    if (this.scheduled === undefined) {
      throw new Error('dsh-plugin-desktop: the Cordis shell plugin did not register a window')
    }
  }

  show(): void {
    this.logInfo('dsh-plugin-desktop: Node Host show() deferred to Wails shell')
  }

  notifyAttention(notification: DesktopNotification): void {
    this.showNotification(notification)
  }

  registerTrayItem(item: DesktopTrayItem): DesktopTrayItemRegistration {
    const key = Symbol('node-tray-item')
    this.trayItems.set(key, item)
    return {
      refresh: () => {},
      dispose: () => { this.trayItems.delete(key) },
    }
  }

  openTerminal(): void {
    this.logInfo('dsh-plugin-desktop: Node Host openTerminal() is unavailable (use Wails CapabilitiesService)')
  }

  reloadRenderer(): void {
    this.logInfo('dsh-plugin-desktop: Node Host reloadRenderer() deferred to Wails shell')
  }

  toggleDeveloperTools(): void {
    this.logInfo('dsh-plugin-desktop: Node Host toggleDeveloperTools() deferred to Wails shell')
  }

  async exportDiagnostics(): Promise<void> {
    this.logInfo('dsh-plugin-desktop: Node Host exportDiagnostics() is not wired; use CLI --export-diagnostics')
  }

  async pickDirectory(): Promise<string | null> {
    this.logInfo('dsh-plugin-desktop: Node Host pickDirectory() deferred to Wails dialogs')
    return null
  }

  openProfileCreateWindow(_options: Omit<ProfileCreateWindowOptions, 'locale'>): void {
    this.logInfo('dsh-plugin-desktop: Node Host profile creator deferred to Wails AuxWindowService')
  }

  async validateDirectory(_path: string): Promise<boolean> {
    return true
  }

  reportRendererBoot(_report: RendererBootReport): void {}

  setLocalePreference(preference: DesktopLocale | undefined): void {
    this.currentLocale = preference ?? desktopLocaleFromLanguageTag(languageTagFromEnv())
  }

  setThemeSource(_source: DesktopThemeSource): void {
    // Native appearance is owned by Wails.
  }

  async requestRestart(): Promise<void> {
    if (this.restartRequest !== undefined) return this.restartRequest
    this.restartRequest = this.restart()
    try {
      await this.restartRequest
    } finally {
      this.restartRequest = undefined
    }
  }

  async requestRecoveryRestart(): Promise<void> {
    if (this.restartRequest !== undefined) return this.restartRequest
    this.restartRequest = this.restart('recovery')
    try {
      await this.restartRequest
    } finally {
      this.restartRequest = undefined
    }
  }

  prepareToQuit(): void {
    this.quitting = true
  }


  /** Electron-compat: Node Host has no renderer health gate. */
  get rendererBootFailureReason(): 'renderer-failed' | 'renderer-timeout' | undefined {
    return undefined
  }

  beginRendererBootMonitoring(): Promise<{ report: { status: 'healthy' } }> {
    return Promise.resolve({ report: { status: 'healthy' as const } })
  }

  stopRendererBootMonitoring(): void {}

  configureTerminal(_spec: unknown): void {}

  /** Test/helper: whether prepareToQuit was called. */
  get isQuitting(): boolean {
    return this.quitting
  }

  /** Test/helper: currently scheduled shell spec, if any. */
  get scheduledSpec(): DesktopShellSpec | undefined {
    return this.scheduled
  }
}
