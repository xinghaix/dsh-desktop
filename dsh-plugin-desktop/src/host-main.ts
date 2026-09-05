/** Node-first Cordis Host for Wails: no Electron app.whenReady / BrowserWindow / Tray. */

import { randomUUID } from 'node:crypto'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  boot,
  installFailLoud,
  loadLayeredEnv,
  PROFILE_PATCH_FILENAME,
  resolveProfileDir,
  type FailLoudProcess,
} from '@deepseek-ai/dsh-app-boot'
import { provideCmdline } from '@deepseek-ai/dsh-cmdline'
import { resolveDshHome } from '@deepseek-ai/dsh-home-paths'
import { DSH_LAUNCH_ENVIRONMENT_KEY } from '@deepseek-ai/dsh-launch-environment'
import type {} from '@deepseek-ai/dsh-web-app'
import type {} from '@deepseek-ai/dsh-client-connection'
import { isDesktopInstallerQuitRequest } from './desktop-installer-quit.ts'
import { createDesktopBrowserAccess } from './desktop-browser-access.ts'
import {
  installDesktopDshRuntime,
  installDesktopPnpmRuntime,
} from './desktop-runtime-environment.ts'
import { NodeDesktopRuntime } from './node-desktop-runtime.ts'
import {
  resolveDesktopAppDataDirectory,
  resolveDesktopHomeDirectory,
  resolveDesktopUserDataDirectory,
} from './host-paths.ts'
import { resolveDesktopElectronVersion } from './peer-electron-version.ts'
import { readFileSync } from 'node:fs'
import { getOrCreateDesktopInstallationId } from './desktop-installation-id.ts'
import {
  ElectronStderrLogger,
  installDesktopUncaughtExceptionLogging,
  type DesktopLogger,
} from './desktop-logger.ts'
import {
  beginDesktopRun,
  type DesktopRun,
} from './crash-evidence.ts'
import { dshProductVersion } from './dsh-product-version.ts'
import { createDesktopLifecycleRecorder } from './lifecycle-events.ts'
import type {
  DesktopLifecycleFailureReason,
  DesktopLifecycleRendererFailureReason,
} from './lifecycle-events.ts'
import { FileExporter } from './file-exporter.ts'
import { desktopNativeCopy } from './native-dialog-copy.ts'
import { DESKTOP_SETTINGS_NAMESPACE, type DesktopSettings } from './index.ts'
import {
  desktopLanBrowserUrls,
  desktopLoopbackBrowserUrl,
} from './desktop-network.ts'
import { desktopLanAddresses } from './lan-addresses.ts'
import {
  createLanHttpsCertificate,
  DesktopLanHttpsCertificateError,
  type DesktopLanHttpsPrivateKeyProtector,
} from './lan-https-certificate.ts'
import { createNodeLanHttpsPrivateKeyProtector } from './node-secret-protector.ts'
import {
  DESKTOP_LAN_HTTPS_CA_PATH,
  DesktopLanHttpsRuntime,
} from './lan-https-runtime.ts'
import { LogFileSink } from './log-files.ts'
import { maskSecrets } from './mask-secrets.ts'
import { resolveDesktopShellEnvironment } from './shell-environment.ts'
import { installProfilePackageResolver } from './module-resolution.ts'
import { packagedDependencyPath } from './packaged-runtime-path.ts'
import {
  beginDesktopProfileStartup,
  assertDesktopProfileName,
  createDesktopWebProfile,
  listDesktopProfiles,
  canDeleteDesktopProfile,
  deleteDesktopProfile,
  readDesktopProfileState,
  selectDesktopProfile,
} from './profile-manager.ts'
import { DesktopProfileService } from './profile-service.ts'
import { DesktopActionsService } from './desktop-actions.ts'
import { clearDesktopProfilePluginState, DesktopPluginsService } from './desktop-plugins.ts'
import {
  desktopMarketSnapshotWithEffective,
  readDesktopMarketStateForUserData,
  selectDesktopMarketProvider,
  type DesktopMarketProvider,
  type DesktopMarketSnapshot,
} from './desktop-market.ts'
import DesktopSettingsController from './desktop-settings-controller.ts'
import {
  clearDesktopProfilePreferences,
  readDesktopProfilePreferences,
  writeDesktopProfilePreferences,
  type DesktopProfilePreferences,
  type DesktopProfilePreferencesStateV1,
} from './profile-preferences.ts'
import {
  DesktopStartupRecoveryController,
  DesktopStartupRecoveryControllerError,
} from './startup-recovery-controller.ts'
import type {
  RecoveryWindowResult,
  DesktopStartupRecoveryConfigurationPaths,
  DesktopStartupRecoveryProfileActions,
  DesktopStartupFailureStage,
} from './startup-recovery-window.ts'
import { routeDesktopStartupFailure } from './startup-failure-routing.ts'
import { DesktopStartupGeneration } from './startup-generation.ts'
import {
  desktopInstallAnchor,
  healDesktopProfileModuleFallback,
  prepareDesktopProfile,
  type SkippedOptionalEntry,
} from './profile.ts'
import { DesktopProfileCheckpoint } from './profile-checkpoint.ts'
import {
  completeOrSkipDesktopSetupWizard,
  desktopSetupWizardRequired,
  desktopSetupWizardStateConstants,
  readDesktopSetupWizardState,
} from './setup-wizard-state.ts'
import {
  migrateDesktopBrowserAccessSettings,
  migrateDesktopWindowMaterialSettings,
  readDesktopSetupWizardSettings,
  updateDesktopSetupWizardSettings,
  type DesktopSetupWizardSettings,
} from './setup-wizard-settings.ts'
import {
  clearDesktopProfileUsageHistory,
  desktopReleaseUserDataLocations,
  inspectDesktopProfileChannelAdmission,
} from './profile-channel-admission.ts'
import {
  formatProfileMaterializationFailure,
  materializeProfile,
  ProfileMaterializationError,
} from './profile-materializer.ts'
import {
  formatRecoveryPluginRemoveFailure,
  removeRecoveryPlugin,
} from './recovery-plugin-uninstall.ts'
import type { DesktopPnpmBootstrap } from './pnpm.ts'
import {
  createDesktopExitCoordinator,
  createDesktopShutdown,
  installShutdownRequests,
  type DesktopShutdown,
} from './shutdown.ts'
import {
  diagnoseWindowsVolumes,
  formatWindowsVolumeConcern,
  type WindowsVolumeConcern,
} from './windows-volume-diagnostics.ts'
import type { RendererBootReport } from './renderer-boot-contract.ts'
import { desktopLocaleFromLanguageTag, desktopTrayLabel } from './tray-locale.ts'
import {
  DESKTOP_NOTIFICATIONS_SETTINGS_NAMESPACE,
  type DesktopNotificationSettings,
} from './notifications.ts'
import {
  desktopDefaultRelaunchArguments,
  desktopRecoveryModeRequested,
  desktopRecoveryRelaunchArguments,
  desktopSafeModeRelaunchArguments,
  desktopSafeModeRequested,
  DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT,
} from './relaunch-arguments.ts'
import {
  announceWailsHostAuthHeader,
  announceWailsHostLanHttps,
  announceWailsHostReady,
  announceWailsHostRecoveryRequired,
  desktopWailsHostSidecarRequested,
  desktopWailsSkipElectronGui,
  DSH_WAILS_HOST_SIDECAR_ENV,
} from './wails-host-sidecar.ts'
import {
  cleanupDesktopSafeModeEnvironment,
  DESKTOP_SAFE_MODE_DEFAULTS,
  DESKTOP_SAFE_MODE_PROFILE_NAME,
  ensureDesktopSafeModeEnvironment,
  resetDesktopSafeModeEnvironment,
  desktopSafeModePaths,
  type DesktopSafeModePaths,
} from './safe-mode.ts'
import {
  recoverOversizedSessionProjectionCache,
  type SessionProjectionCacheRecoveryResult,
} from './session-projcache-recovery.ts'
import {
  DESKTOP_PACKAGE_NAME,
  DESKTOP_PRODUCT_NAME,
  DESKTOP_RELEASE_CHANNEL,
} from './product-identity.ts'
import { desktopRecoveryCopy } from './recovery-copy.ts'

const BIN_NAME = DESKTOP_PACKAGE_NAME
const PRODUCT_NAME = DESKTOP_PRODUCT_NAME

function desktopProductVersion(moduleUrl: string = import.meta.url): string {
  const value: unknown = JSON.parse(readFileSync(new URL('../package.json', moduleUrl), 'utf8'))
  if (value === null || typeof value !== 'object' || typeof (value as { version?: unknown }).version !== 'string') {
    throw new Error('dsh-plugin-desktop: package.json has no product version')
  }
  return (value as { version: string }).version
}


/** Node Host: AES-GCM file-backed protector (see node-secret-protector threat model). */
function desktopLanHttpsPrivateKeyProtector(): DesktopLanHttpsPrivateKeyProtector {
  return createNodeLanHttpsPrivateKeyProtector(resolveDesktopUserDataDirectory())
}

class RendererStartupFailure extends Error {
  constructor(
    readonly reason: 'renderer-failed' | 'renderer-timeout',
    report: Extract<RendererBootReport, { status: 'failed' }>,
  ) {
    super(report.error ?? `Renderer boot failed for ${String(report.plugins.length)} plugin(s)`)
    this.name = 'RendererStartupFailure'
  }
}

function lifecycleRendererFailureReason(
  reason: 'renderer-failed' | 'renderer-timeout' | undefined,
): DesktopLifecycleRendererFailureReason {
  return reason === 'renderer-timeout' ? 'renderer-timeout' : 'renderer-failed'
}

function lifecycleStartupFailureReason(
  cause: unknown,
  runtime: NodeDesktopRuntime,
): DesktopLifecycleFailureReason {
  if (cause instanceof RendererStartupFailure) return cause.reason
  return runtime.rendererBootFailureReason ?? 'startup-failed'
}

/** Report optional user UI plugins skipped to keep startup recoverable. */
function notifySkippedOptionalEntries(
  runtime: NodeDesktopRuntime,
  logger: DesktopLogger,
  entries: readonly SkippedOptionalEntry[],
): void {
  if (entries.length === 0) return
  const copy = desktopNativeCopy(runtime.locale)
  const names = entries.map(entry => entry.name)
  try {
    runtime.updates.notify({
      title: copy.skippedPluginTitle,
      body: copy.skippedPluginBody(names[0]!, names.length - 1),
    })
  } catch (cause) {
    logger.error(`${BIN_NAME}: failed to show skipped plugin notification: ${cause instanceof Error ? cause.message : String(cause)}`)
  }
}

/** Surface path/volume risks that otherwise become obscure sandbox or pnpm failures later. */
function warnWindowsVolumeConcerns(logger: DesktopLogger, concerns: readonly WindowsVolumeConcern[]): void {
  for (const concern of concerns) {
    logger.error(`${BIN_NAME}: Windows volume warning: ${formatWindowsVolumeConcern(concern)}`)
  }
}

/** Notify once after the UI is ready; stderr carries the exact paths. */
function notifyWindowsVolumeConcerns(
  runtime: NodeDesktopRuntime,
  logger: DesktopLogger,
  concerns: readonly WindowsVolumeConcern[],
): void {
  if (concerns.length === 0) return
  const copy = desktopNativeCopy(runtime.locale)
  const concernLabel = concerns[0]?.label
  const label = runtime.locale === 'zh'
    ? concernLabel === 'application install' ? '应用安装目录'
      : concernLabel === 'desktop user data' ? '桌面用户数据'
        : concernLabel === 'DSH home' ? 'DSH 主目录'
          : '某个配置路径'
    : concernLabel ?? 'A configured path'
  try {
    runtime.updates.notify({
      title: copy.unsupportedStorageTitle,
      body: copy.unsupportedStorageBody(label),
    })
  } catch (cause) {
    logger.error(`${BIN_NAME}: failed to show Windows volume warning: ${cause instanceof Error ? cause.message : String(cause)}`)
  }
}

function notifySessionProjectionCacheRecovery(
  runtime: NodeDesktopRuntime,
  logger: DesktopLogger,
  _recovery: Extract<SessionProjectionCacheRecoveryResult, { status: 'quarantined' }>,
): void {
  try {
    runtime.updates.notify({
      title: 'Recovered Session Cache',
      body: 'An oversized session projection cache was moved aside and will be rebuilt from session history.',
    })
  } catch (cause) {
    logger.error(`${BIN_NAME}: failed to show session projection cache recovery notification: ${cause instanceof Error ? cause.message : String(cause)}`)
  }
}

/** Explain the disposable environment after the Safe Mode renderer is healthy. */
function notifyDesktopSafeModeActive(
  runtime: NodeDesktopRuntime,
  logger: DesktopLogger,
): void {
  const copy = desktopRecoveryCopy(runtime.locale)
  try {
    runtime.updates.notify({
      title: copy.safeModeNotificationTitle,
      body: copy.safeModeNotificationBody,
    })
  } catch (cause) {
    logger.error(`${BIN_NAME}: failed to show Safe Mode notification: ${cause instanceof Error ? cause.message : String(cause)}`)
  }
}

/** Use one Profile-owned Market request as the first composition input. */
function desktopProfileMarketSnapshot(market: DesktopMarketProvider): DesktopMarketSnapshot {
  return Object.freeze({
    requested: market,
    effective: market,
    legacyDefaulted: false,
  })
}

/** Project exactly the first-stage Profile fields from an effective settings view. */
function desktopProfilePreferencesFromSettings(
  desktop: Pick<DesktopSettings, 'mode' | 'openBrowser' | 'networkExposure'>,
  notifications: Readonly<DesktopNotificationSettings>,
  market: DesktopMarketProvider,
): DesktopProfilePreferences {
  return Object.freeze({
    mode: desktop.mode,
    openBrowser: desktop.openBrowser,
    networkExposure: desktop.networkExposure,
    notifications: Object.freeze({ ...notifications }),
    market,
  })
}

/** Preserve device-shared Wizard fields while mirroring one Profile's leaves. */
function setupSettingsWithProfilePreferences(
  current: DesktopSetupWizardSettings,
  preferences: DesktopProfilePreferences,
): DesktopSetupWizardSettings {
  return Object.freeze({
    ...current,
    mode: preferences.mode,
    openBrowser: preferences.openBrowser,
    networkExposure: preferences.networkExposure,
    notifications: Object.freeze({ ...preferences.notifications }),
  })
}

/** Mirror only the Profile-owned settings leaves into the exact prepared document. */
async function mirrorDesktopProfilePreferences(
  settingsDocument: string,
  preferences: DesktopProfilePreferences,
): Promise<void> {
  const current = readDesktopSetupWizardSettings(settingsDocument)
  await updateDesktopSetupWizardSettings(
    settingsDocument,
    setupSettingsWithProfilePreferences(current, preferences),
  )
}

/** Start Cordis Host without Electron GUI lifecycle (Wails / Node Host). */
async function start(): Promise<void> {
  // Prefer Node Host announce protocol; keep argv marker for relaunch filtering.
  process.env[DSH_WAILS_HOST_SIDECAR_ENV] = '1'
  if (!process.argv.includes(DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT)) {
    process.argv.push(DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT)
  }
  if (isDesktopInstallerQuitRequest(process.argv, process.platform)) {
    process.exitCode = 0
    return
  }

  let shutdown: DesktopShutdown | undefined
  let removeShutdownRequests: (() => void) | undefined
  let removeUncaughtExceptionLogging: (() => void) | undefined
  let removeChildProcessLogging: (() => void) | undefined
  let fileExporter: FileExporter | undefined
  let runtime!: NodeDesktopRuntime
  let logSink: LogFileSink | undefined
  let startupRecoveryController: DesktopStartupRecoveryController | undefined
  let startupRecoveryWindow: { show(): void } | undefined
  let setupWizardWindow: { show(): void } | undefined
  let profileCompatibilityCreateWindow: { open(): void } | undefined
  let profileSelectionWindow: { show(): void } | undefined
  let startupRecoveryConfigurationPaths: DesktopStartupRecoveryConfigurationPaths | undefined
  let profileCheckpoint: DesktopProfileCheckpoint | undefined
  let startupRecoveryProfileActions: DesktopStartupRecoveryProfileActions | undefined
  let safeModePaths: DesktopSafeModePaths | undefined
  let prepareSafeMode: (() => void) | undefined
  let sessionProjectionCacheRecovery:
    | Extract<SessionProjectionCacheRecoveryResult, { status: 'quarantined' }>
    | undefined
  let recoveryTerminalAvailable = false
  let startupStage: DesktopStartupFailureStage = 'electron-ready'
  const desktopUserDataDir = resolveDesktopUserDataDirectory()
  const appVersion = desktopProductVersion()
  const currentDshVersion = dshProductVersion()
  const setupWizardVersions = Object.freeze({
    desktopVersion: appVersion,
    dshVersion: currentDshVersion,
    setupRevision: desktopSetupWizardStateConstants.setupRevision,
  })
  const recoveryModeRequested = desktopRecoveryModeRequested()
  const wailsElectronLight = desktopWailsSkipElectronGui()
  const safeModeRequested = desktopSafeModeRequested()
  const inheritedDshHome = process.env.DSH_HOME
  const safeModeHomeDir = desktopSafeModePaths(desktopUserDataDir).homeDir
  try {
    logSink = new LogFileSink(join(desktopUserDataDir, 'logs'), {
      maxFileBytes: 10 * 1024 * 1024,
      maxDirectoryBytes: 200 * 1024 * 1024,
    })
    logSink.enforceDirectoryCap()
    logSink.purgeOlderThan(7)
    logSink.writeHeader(`--- ${BIN_NAME} ${PRODUCT_NAME} ${appVersion} ${process.platform} node ${process.version} run ${Date.now()} ---`)
  } catch (cause) {
    const detail = cause instanceof Error ? cause.message : String(cause)
    process.stderr.write(`${BIN_NAME}: file logging unavailable: ${maskSecrets(detail)}\n`)
    logSink = undefined
  }
  const electronLogger = new ElectronStderrLogger(logSink)
  if (safeModeRequested) {
    safeModePaths = ensureDesktopSafeModeEnvironment(desktopUserDataDir)
  } else {
    if (process.env.DSH_HOME === safeModeHomeDir) delete process.env.DSH_HOME
    try {
      cleanupDesktopSafeModeEnvironment(desktopUserDataDir)
    } catch (cause) {
      electronLogger.error(`${BIN_NAME}: failed to remove the previous Safe Mode environment: ${cause instanceof Error ? cause.message : String(cause)}`)
    }
  }
  const generation = new DesktopStartupGeneration({ logger: electronLogger })
  const generationId = generation.id
  const lifecycleRecorder = createDesktopLifecycleRecorder({
    userDataDir: desktopUserDataDir,
    appVersion,
    platform: process.platform,
    arch: process.arch,
    logger: electronLogger,
  })
  lifecycleRecorder.startStartup(startupStage)
  // Node Host: crashReporter is Electron-only; skipped.
  let desktopRun: DesktopRun | undefined
  try {
    desktopRun = beginDesktopRun(
      join(desktopUserDataDir, 'crash-evidence', 'active-run.json'),
      {
        startedAt: new Date().toISOString(),
        pid: process.pid,
        version: appVersion,
      },
    )
    const previousRun = desktopRun.previousRun
    if (previousRun !== undefined) {
      electronLogger.error('unreadable' in previousRun
        ? `${BIN_NAME}: previous desktop run did not shut down cleanly (active run marker unreadable)`
        : `${BIN_NAME}: previous desktop run did not shut down cleanly (startedAt: ${previousRun.startedAt}, pid: ${String(previousRun.pid)}, version: ${previousRun.version})`)
    }
  } catch (cause) {
    electronLogger.error(`${BIN_NAME}: active run tracking unavailable: ${cause instanceof Error ? cause.message : String(cause)}`)
  }
  // Node Host: no Electron child_process logging hook (Electron main installs one).
  const nativeExit = createDesktopExitCoordinator(
    {
      prepareToQuit: () => { runtime.prepareToQuit() },
      relaunch: args => {
        // Node Host cannot Electron-relaunch; exit and let Wails HostSidecar respawn.
        process.stderr.write(`${BIN_NAME}: Node Host relaunch requested; exiting for Wails respawn\n`)
        void args
        process.exitCode = 0
      },
      exit: code => { process.exit(code) },
    },
    () => {
      removeShutdownRequests?.()
      removeUncaughtExceptionLogging?.()
      removeChildProcessLogging?.()
      if (safeModePaths !== undefined) {
        if (inheritedDshHome === undefined) delete process.env.DSH_HOME
        else process.env.DSH_HOME = inheritedDshHome
        try {
          cleanupDesktopSafeModeEnvironment(desktopUserDataDir)
        } catch (cause) {
          electronLogger.error(`${BIN_NAME}: failed to remove the Safe Mode environment: ${cause instanceof Error ? cause.message : String(cause)}`)
        }
      }
      try {
        desktopRun?.markClean()
      } catch (cause) {
        electronLogger.error(`${BIN_NAME}: failed to clear active run marker: ${cause instanceof Error ? cause.message : String(cause)}`)
      }
    },
  )
  let restartRequested = false
  const installationId = await getOrCreateDesktopInstallationId(desktopUserDataDir)
  runtime = new NodeDesktopRuntime(async target => {
    if (shutdown === undefined) {
      throw new Error('dsh-plugin-desktop: shutdown coordinator is not ready')
    }
    if (restartRequested) return
    restartRequested = true
    nativeExit.requestRelaunch(
      target === 'recovery'
        ? desktopRecoveryRelaunchArguments()
        : desktopDefaultRelaunchArguments(),
    )
    await shutdown.request(0)
  }, electronLogger, desktopUserDataDir, installationId)
  const finalExit = (code: number): void => { nativeExit.finish(code) }
  shutdown = createDesktopShutdown(
    async () => { await generation.release() },
    finalExit,
  )
  const requestQuit = (code: number): void => { void shutdown.request(code) }
  removeUncaughtExceptionLogging = installDesktopUncaughtExceptionLogging(
    process,
    electronLogger,
    requestQuit,
    { evidenceDir: join(desktopUserDataDir, 'crash-evidence') },
  )
  const nodeQuitSource = {
    on(_event: 'before-quit' | 'window-all-closed', _listener: (..._args: never[]) => void) { return this },
    off(_event: 'before-quit' | 'window-all-closed', _listener: (..._args: never[]) => void) { return this },
  }
  removeShutdownRequests = installShutdownRequests(process, nodeQuitSource, requestQuit)

  const openStartupRecoveryWindow = async (
    failureDetail: string,
    _controller: DesktopStartupRecoveryController | undefined,
    _requested = false,
  ): Promise<RecoveryWindowResult | 'unavailable'> => {
    // Keep Node Host recovery affordances live for parity with Electron main; Wails owns UI.
    const deferredRecoveryCapabilities = {
      terminalAvailable: recoveryTerminalAvailable,
      profileActions: startupRecoveryProfileActions,
      enterSafeMode: prepareSafeMode,
    }
    void deferredRecoveryCapabilities
    announceWailsHostRecoveryRequired(maskSecrets(failureDetail))
    electronLogger.error(
      `${BIN_NAME}: Node Host recovery UI deferred to Wails (${maskSecrets(failureDetail)})`,
    )
    return 'unavailable'
  }

  const showPreHostSurface = (): boolean => {
    if (profileCompatibilityCreateWindow !== undefined) {
      profileCompatibilityCreateWindow.open()
      return true
    }
    if (profileSelectionWindow !== undefined) {
      profileSelectionWindow.show()
      return true
    }
    if (setupWizardWindow !== undefined) {
      setupWizardWindow.show()
      return true
    }
    if (startupRecoveryWindow !== undefined) {
      startupRecoveryWindow.show()
      return true
    }
    return false
  }
  // Node Host: no Electron activate/second-instance / whenReady.
  void showPreHostSurface
  try {
    startupStage = 'shell-environment'
    lifecycleRecorder.transitionStartupStage(startupStage)
    const shellEnvironmentResolution = await resolveDesktopShellEnvironment({
      environment: process.env,
      home: resolveDesktopHomeDirectory(),
      isPackaged: false,
      platform: process.platform,
    })
    for (const [name, value] of Object.entries(shellEnvironmentResolution.updates)) process.env[name] = value
    const profileUserDataDir = safeModePaths?.userDataDir ?? desktopUserDataDir
    const homeDir = safeModePaths?.homeDir ?? resolveDshHome()
    if (safeModePaths !== undefined) process.env.DSH_HOME = homeDir
    prepareSafeMode = safeModePaths === undefined
      ? () => {
          const paths = resetDesktopSafeModeEnvironment(desktopUserDataDir)
          try {
            createDesktopWebProfile(paths.homeDir, DESKTOP_SAFE_MODE_PROFILE_NAME)
            selectDesktopProfile(
              join(paths.userDataDir, 'profile-selection', 'state.json'),
              paths.homeDir,
              DESKTOP_SAFE_MODE_PROFILE_NAME,
            )
          } catch (cause) {
            cleanupDesktopSafeModeEnvironment(desktopUserDataDir)
            throw cause
          }
        }
      : undefined
    const projectionCacheRecovery = recoverOversizedSessionProjectionCache(homeDir)
    if (projectionCacheRecovery.status === 'quarantined') {
      sessionProjectionCacheRecovery = projectionCacheRecovery
      electronLogger.error(
        `${BIN_NAME}: quarantined oversized session projection cache (${String(projectionCacheRecovery.sizeBytes)} bytes) at `
          + `${projectionCacheRecovery.cachePath}; backup saved to ${projectionCacheRecovery.backupPath}`,
      )
    }
    const windowsVolumeConcerns = diagnoseWindowsVolumes(process.platform, [
      { label: 'application install', path: process.execPath },
      { label: 'desktop user data', path: desktopUserDataDir },
      { label: 'DSH home', path: homeDir },
    ])
    warnWindowsVolumeConcerns(electronLogger, windowsVolumeConcerns)

    const failLoudProcess: FailLoudProcess = {
      on: (event, handler) => process.on(event, handler),
      off: (event, handler) => process.off(event, handler),
      stderr: electronLogger,
      exit: finalExit,
    }
    installFailLoud(BIN_NAME, failLoudProcess, async () => { await generation.release() })

    startupStage = 'runtime-bootstrap'
    lifecycleRecorder.transitionStartupStage(startupStage)
    const environment = loadLayeredEnv(BIN_NAME, process.cwd())
    const electronVersion = resolveDesktopElectronVersion()
    if (process.versions.electron === undefined) {
      electronLogger.error(
        `${BIN_NAME}: Node Host using peer Electron version ${electronVersion} for npm_config_target (process.versions.electron unset)`,
      )
    }
    const pnpmBinPath = packagedDependencyPath(import.meta.url, 'pnpm/bin/pnpm.mjs')
    const pnpmRuntime = installDesktopPnpmRuntime({
      platform: process.platform,
      appExecutable: process.execPath,
      pnpmBinPath,
      electronVersion,
      stateDir: join(desktopUserDataDir, 'runtime-commands'),
      environment: process.env,
    })
    const dshBootstrapPath = fileURLToPath(new URL('./desktop-cli.js', import.meta.url))
    const releasePnpmRuntime = generation.own(() => { pnpmRuntime.dispose() })
    const selectionStatePath = join(profileUserDataDir, 'profile-selection', 'state.json')
    const pluginManagementStatePath = join(profileUserDataDir, 'plugin-management', 'state.json')
    const marketUserDataDir = profileUserDataDir
    const releaseUserDataLocations = desktopReleaseUserDataLocations(
      resolveDesktopAppDataDirectory(),
      marketUserDataDir,
    )
    const createFreshDesktopProfile = (name: string) => {
      const created = createDesktopWebProfile(homeDir, name)
      clearDesktopProfileUsageHistory(releaseUserDataLocations, created.dir)
      return created
    }
    startupStage = 'profile-selection'
    lifecycleRecorder.transitionStartupStage(startupStage)
    const profileDirectoriesBeforeStartup = new Set(
      listDesktopProfiles(homeDir).map(profile => profile.dir),
    )
    // Keep Profile recovery usable when the persisted selection no longer exists.
    // Locale is retained for Electron-parity dialog paths; Wails owns hybrid UX copy.
    const locale = desktopLocaleFromLanguageTag(process.env.LANG ?? 'en')
    void locale
    const recoveryProfileToken = randomUUID()
    let activeProfileName = readDesktopProfileState(selectionStatePath).active
    let expectedRecoveryProfileName = activeProfileName
    const openStartupProfileCreator = async (): Promise<void> => {
      announceWailsHostRecoveryRequired('profile-create-requested')
      electronLogger.error(`${BIN_NAME}: Node Host profile creator deferred to Wails AuxWindowService`)
    }
    startupRecoveryProfileActions = {
      token: recoveryProfileToken,
      list: () => {
        const selectedProfileName = readDesktopProfileState(selectionStatePath).active
        return listDesktopProfiles(homeDir).map(profile => ({
          name: profile.name,
          current: profile.name === selectedProfileName,
          selectable: profile.webCapable && profile.problem === undefined,
        }))
      },
      switchProfile: (name, token) => {
        if (token !== recoveryProfileToken) {
          throw new Error(`${BIN_NAME}: the Profile recovery action is no longer valid`)
        }
        assertDesktopProfileName(name)
        const selection = readDesktopProfileState(selectionStatePath)
        if (selection.active !== expectedRecoveryProfileName) {
          throw new Error(`${BIN_NAME}: active Profile changed outside recovery`)
        }
        const target = listDesktopProfiles(homeDir).find(profile => profile.name === name)
        if (target === undefined || !target.webCapable || target.problem !== undefined) {
          throw new Error(`${BIN_NAME}: Profile ${JSON.stringify(name)} is unavailable`)
        }
        selectDesktopProfile(selectionStatePath, homeDir, name)
        expectedRecoveryProfileName = name
      },
      openCreator: openStartupProfileCreator,
    }
    const profileStartup = beginDesktopProfileStartup(selectionStatePath, homeDir)
    activeProfileName = profileStartup.profileName
    expectedRecoveryProfileName = activeProfileName
    const activeProfileDir = resolveProfileDir(activeProfileName, homeDir)
    if (!profileDirectoriesBeforeStartup.has(activeProfileDir)) {
      clearDesktopProfileUsageHistory(releaseUserDataLocations, activeProfileDir)
    }
    // Recovery can open before Profile composition and Host boot. Fix the
    // launcher-owned terminal identity as soon as Profile selection succeeds
    // so every recovery entry path exposes the same terminal action.
    runtime.configureTerminal({
      profileName: activeProfileName,
      profileDir: activeProfileDir,
      homeDir,
    })
    recoveryTerminalAvailable = true
    if (safeModePaths !== undefined) {
      const safeModeTray = runtime.registerTrayItem({
        group: 'status',
        order: -100,
        label: () => desktopTrayLabel(runtime.locale, 'exitSafeMode'),
        invoke: async () => {
          if (restartRequested) return
          restartRequested = true
          nativeExit.requestRelaunch(desktopDefaultRelaunchArguments())
          await shutdown.request(0)
        },
      })
      generation.own(() => { safeModeTray.dispose() })
    }
    if (!recoveryModeRequested) {
      while (true) {
        const admission = inspectDesktopProfileChannelAdmission(
          releaseUserDataLocations,
          activeProfileDir,
          activeProfileName,
        )
        if (admission.status === 'allow') break
        if (wailsElectronLight) {
          // Electron-light Host: no BrowserWindow dialogs; allow and continue so
          // Wails can own profile UX. Log the cross-channel admission warning.
          electronLogger.error(
            `${BIN_NAME}: Wails Host sidecar auto-allowing cross-channel Profile ${JSON.stringify(activeProfileName)} (Electron dialogs skipped)`,
          )
          break
        }
        // Unreachable when Node Host forces Wails sidecar markers above.
        throw new Error(`${BIN_NAME}: Node Host must not open Electron profile-compatibility dialogs`)
      }
    }
    try {
      profileCheckpoint = new DesktopProfileCheckpoint({
        userDataDir: profileUserDataDir,
        profileDir: activeProfileDir,
        homeDir,
        profileName: activeProfileName,
        provider: 'desktop-profile',
        appVersion,
        desktopPackageName: DESKTOP_PACKAGE_NAME,
        releaseChannel: DESKTOP_RELEASE_CHANNEL,
        dshVersion: currentDshVersion,
      })
    } catch (cause) {
      electronLogger.error(
        `${BIN_NAME}: healthy profile checkpoints are unavailable: ${cause instanceof Error ? cause.message : String(cause)}`,
      )
    }
    startupRecoveryConfigurationPaths = {
      settingsDocument: join(homeDir, 'settings.yaml'),
      profilePatch: join(activeProfileDir, PROFILE_PATCH_FILENAME),
      profileManifest: join(activeProfileDir, 'package.json'),
      profileDirectory: activeProfileDir,
    }
    if (profileCheckpoint !== undefined) {
      startupRecoveryController = new DesktopStartupRecoveryController({
        pluginState: {
          profileName: activeProfileName,
          homeDir,
          statePath: pluginManagementStatePath,
        },
        generationId,
        currentGeneration: () => ({
          profileName: readDesktopProfileState(selectionStatePath).active,
          generationId,
        }),
        uninstallPlugin: async packageName => {
          try {
            await removeRecoveryPlugin({
              appExecutable: process.execPath,
              dshBootstrapPath,
              profileName: activeProfileName,
              profileDir: activeProfileDir,
              homeDir,
              nodeBinDir: pnpmRuntime.nodeBinDir,
              nodeShimPath: pnpmRuntime.nodeShimPath,
              pnpmBinDir: pnpmRuntime.pathDir,
              electronVersion,
              packageName,
            })
          } catch (cause) {
            const detail = maskSecrets(formatRecoveryPluginRemoveFailure(cause))
            electronLogger.error(`${BIN_NAME}: recovery plugin uninstall failed:\n${detail}`)
            throw new DesktopStartupRecoveryControllerError(
              'operation-failed',
              'The plugin could not be removed from the current Profile.',
              { operationStage: 'plugin-change', diagnosticDetail: detail },
            )
          }
        },
        checkpoints: profileCheckpoint,
        openCheckpointDirectory: async path => {
          process.stderr.write(`${BIN_NAME}: Node Host cannot openPath(${path}); use Wails reveal\n`)
        },
        afterCheckpointRestore: async result => {
          if (!result.dependencyMaterializationRequired) return
          try {
            await materializeProfile({
              appExecutable: process.execPath,
              clearEnvironmentPath: pnpmRuntime.clearEnvironmentPath,
              pnpmBinPath,
              nodeBinDir: pnpmRuntime.nodeBinDir,
              nodeShimPath: pnpmRuntime.nodeShimPath,
              homeDir,
              profileDir: activeProfileDir,
              electronVersion,
            })
          } catch (cause) {
            const detail = maskSecrets(formatProfileMaterializationFailure(cause))
            electronLogger.error(`${BIN_NAME}: checkpoint dependency materialization failed:\n${detail}`)
            throw new DesktopStartupRecoveryControllerError(
              'operation-failed',
              'The checkpoint files were restored, but Profile dependencies could not be rebuilt.',
              {
                operationStage: 'dependency-materialization',
                diagnosticDetail: detail,
              },
            )
          }
        },
      })
    }
    if (recoveryModeRequested) {
      if (wailsElectronLight) {
        announceWailsHostRecoveryRequired(
          'Recovery mode was requested; open Wails Recovery Assistant (Electron BrowserWindow skipped).',
        )
        electronLogger.error(
          `${BIN_NAME}: Wails Host sidecar recovery requested — Electron recovery window skipped; announced to Wails shell`,
        )
        startupRecoveryController?.dispose()
        startupRecoveryController = undefined
        await shutdown.request(1)
        return
      }
      const recoveryResult = await openStartupRecoveryWindow(
        'Recovery mode was requested from the Desktop restart menu.',
        startupRecoveryController,
        true,
      )
      startupRecoveryController?.dispose()
      startupRecoveryController = undefined
      if (recoveryResult === 'restart') nativeExit.requestRelaunch()
      else if (recoveryResult === 'safe-mode') {
        nativeExit.requestRelaunch(desktopSafeModeRelaunchArguments())
      }
      await shutdown.request(recoveryResult === 'restart' || recoveryResult === 'safe-mode' ? 0 : 1)
      return
    }
    startupStage = 'profile-composition'
    lifecycleRecorder.transitionStartupStage(startupStage)
    const lanAddresses = desktopLanAddresses()
    const legacyMarketSelection = readDesktopMarketStateForUserData(marketUserDataDir)
    let profilePreferences = readDesktopProfilePreferences(marketUserDataDir, activeProfileDir)
    let marketSelection = profilePreferences === undefined
      ? legacyMarketSelection
      : desktopProfileMarketSnapshot(profilePreferences.market)
    const preparationHooks = {
      lanAddresses,
      onSettingsDocumentResolved: (settingsDocument: string) => {
        if (startupRecoveryConfigurationPaths === undefined) return
        startupRecoveryConfigurationPaths = {
          ...startupRecoveryConfigurationPaths,
          settingsDocument,
        }
      },
    }
    await healDesktopProfileModuleFallback(homeDir)
    let prepared = prepareDesktopProfile(
      process.env.DSH_TELEMETRY_DISABLED,
      homeDir,
      process.platform,
      activeProfileName,
      pluginManagementStatePath,
      marketSelection,
      preparationHooks,
    )
    if (safeModePaths !== undefined) {
      const safeModeDefaults = DESKTOP_SAFE_MODE_DEFAULTS
      await updateDesktopSetupWizardSettings(prepared.settingsDocument, safeModeDefaults.settings)
      await selectDesktopMarketProvider(marketUserDataDir, safeModeDefaults.market)
      marketSelection = readDesktopMarketStateForUserData(marketUserDataDir)
      profilePreferences = await writeDesktopProfilePreferences(
        marketUserDataDir,
        prepared.profile.dir,
        desktopProfilePreferencesFromSettings(
          safeModeDefaults.settings,
          safeModeDefaults.settings.notifications,
          safeModeDefaults.market,
        ),
      )
      prepared = prepareDesktopProfile(
        process.env.DSH_TELEMETRY_DISABLED,
        homeDir,
        process.platform,
        activeProfileName,
        pluginManagementStatePath,
        marketSelection,
        preparationHooks,
      )
    } else if (profilePreferences === undefined) {
      const browserAccessMigrated = await migrateDesktopBrowserAccessSettings(prepared.settingsDocument)
      let windowMaterialMigrated = false
      try {
        windowMaterialMigrated = await migrateDesktopWindowMaterialSettings(prepared.settingsDocument)
      } catch (cause) {
        // Legacy Acrylic is already normalized to off by the read boundary. A
        // read-only settings file must not turn removal of the effect into a
        // startup failure merely because the durable cleanup could not be saved.
        electronLogger.error(
          `${BIN_NAME}: failed to persist removed Acrylic material migration: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      }
      if (browserAccessMigrated || windowMaterialMigrated) {
        prepared = prepareDesktopProfile(
          process.env.DSH_TELEMETRY_DISABLED,
          homeDir,
          process.platform,
          activeProfileName,
          pluginManagementStatePath,
          marketSelection,
          preparationHooks,
        )
      }
      const importedSettings = readDesktopSetupWizardSettings(prepared.settingsDocument)
      profilePreferences = await writeDesktopProfilePreferences(
        marketUserDataDir,
        prepared.profile.dir,
        desktopProfilePreferencesFromSettings(
          prepared,
          importedSettings.notifications,
          legacyMarketSelection.requested,
        ),
      )
    } else {
      // Existing Profile state is the source of truth. Mirror it only after the
      // first prepare has resolved this Profile's exact settings document.
      await mirrorDesktopProfilePreferences(prepared.settingsDocument, profilePreferences)
      try {
        await migrateDesktopWindowMaterialSettings(prepared.settingsDocument)
      } catch (cause) {
        // Keep retrying the device-owned Acrylic cleanup on later launches,
        // even after this Profile has completed its one-time preference import.
        electronLogger.error(
          `${BIN_NAME}: failed to persist removed Acrylic material migration: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      }
      await selectDesktopMarketProvider(marketUserDataDir, profilePreferences.market)
      marketSelection = readDesktopMarketStateForUserData(marketUserDataDir)
      prepared = prepareDesktopProfile(
        process.env.DSH_TELEMETRY_DISABLED,
        homeDir,
        process.platform,
        activeProfileName,
        pluginManagementStatePath,
        marketSelection,
        preparationHooks,
      )
    }
    // Safe Mode must reach the working surface with shipped defaults. Its
    // disposable Desktop state deliberately has no Setup marker, so reading
    // it as an ordinary Profile would incorrectly open first-run Setup.
    const setupWizardState = safeModePaths === undefined
      ? readDesktopSetupWizardState(marketUserDataDir, prepared.profile.dir)
      : undefined
    if (safeModePaths === undefined && desktopSetupWizardRequired(setupWizardState, setupWizardVersions)) {
      if (wailsElectronLight) {
        // Electron-light Host: skip BrowserWindow Setup Wizard; mark skipped so
        // subsequent boots continue. Wails AuxWindowService owns hybrid setup UX.
        electronLogger.error(
          `${BIN_NAME}: Wails Host sidecar auto-skipping Desktop Setup Wizard (Electron window skipped)`,
        )
        await completeOrSkipDesktopSetupWizard(
          marketUserDataDir,
          prepared.profile.dir,
          'skipped',
          setupWizardVersions,
        )
      } else {
        throw new Error(`${BIN_NAME}: Node Host must not open Electron Setup Wizard`)
      }
    }
    if (profileCheckpoint === undefined) {
      try {
        profileCheckpoint = new DesktopProfileCheckpoint({
          userDataDir: profileUserDataDir,
          profileDir: prepared.profile.dir,
          homeDir,
          profileName: activeProfileName,
          provider: 'desktop-profile',
          appVersion,
          desktopPackageName: DESKTOP_PACKAGE_NAME,
          releaseChannel: DESKTOP_RELEASE_CHANNEL,
          dshVersion: currentDshVersion,
        })
      } catch (cause) {
        electronLogger.error(
          `${BIN_NAME}: healthy profile checkpoints remain unavailable: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      }
    }
    startupStage = 'runtime-bootstrap'
    lifecycleRecorder.transitionStartupStage(startupStage)
    const dshRuntime = process.platform === 'win32'
      ? installDesktopDshRuntime({
          platform: process.platform,
          appExecutable: process.execPath,
          dshBootstrapPath,
          profileName: activeProfileName,
          homeDir,
          stateDir: join(profileUserDataDir, 'host-commands', activeProfileName),
          environment: process.env,
        })
      : undefined
    const releaseDshRuntime = generation.own(() => { dshRuntime?.dispose() })
    if (prepared.requiresDependencyMigration) {
      electronLogger.error(`${BIN_NAME}: migrating legacy Profile dependency layout with packaged pnpm`)
      try {
        await materializeProfile({
          appExecutable: process.execPath,
          clearEnvironmentPath: pnpmRuntime.clearEnvironmentPath,
          pnpmBinPath,
          nodeBinDir: pnpmRuntime.nodeBinDir,
          nodeShimPath: pnpmRuntime.nodeShimPath,
          homeDir,
          profileDir: prepared.profile.dir,
          electronVersion,
          updateLockfile: true,
        })
        prepared = prepareDesktopProfile(
          process.env.DSH_TELEMETRY_DISABLED,
          homeDir,
          process.platform,
          activeProfileName,
          pluginManagementStatePath,
          marketSelection,
          preparationHooks,
        )
        if (prepared.requiresDependencyMigration) {
          throw new Error(`${BIN_NAME}: packaged pnpm did not produce compatible Profile dependency metadata`)
        }
      } catch (migrationCause) {
        const detail = migrationCause instanceof ProfileMaterializationError
          ? migrationCause.result?.stderr || migrationCause.message
          : migrationCause instanceof Error ? migrationCause.message : String(migrationCause)
        throw new Error(`${BIN_NAME}: Profile dependency migration failed: ${maskSecrets(detail)}`)
      }
    }
    if (prepared.marketFailure !== undefined) {
      electronLogger.error(
        `${BIN_NAME}: requested Market provider ${prepared.market.requested} was disabled for this generation: ${prepared.marketFailure}`,
      )
    }
    let lanHttpsCertificate: Awaited<ReturnType<typeof createLanHttpsCertificate>> | undefined
    let lanHttpsFailureCode: string | undefined
    if (prepared.lanAddresses.length === 0) {
      lanHttpsFailureCode = 'no-address'
    } else {
      try {
        lanHttpsCertificate = await createLanHttpsCertificate(
          marketUserDataDir,
          prepared.lanAddresses,
          desktopLanHttpsPrivateKeyProtector(),
        )
      } catch (cause) {
        lanHttpsFailureCode = cause instanceof DesktopLanHttpsCertificateError
          ? cause.code
          : 'certificate-state'
        electronLogger.error(
          `${BIN_NAME}: LAN HTTPS certificate setup is unavailable: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      }
    }
    const lanHttps = new DesktopLanHttpsRuntime({
      addresses: prepared.lanAddresses,
      ...(lanHttpsCertificate === undefined ? {} : { certificate: lanHttpsCertificate }),
      ...(lanHttpsFailureCode === undefined ? {} : { failureCode: lanHttpsFailureCode }),
      requestedPort: 0,
    })
    const browserAccess = createDesktopBrowserAccess(
      prepared.mode === 'compatibility' && prepared.openBrowser,
    )
    const desktopPnpmBootstrap: DesktopPnpmBootstrap = {
      activeProfileName,
      activeProfileDir: prepared.profile.dir,
      homeDir,
      appExecutable: process.execPath,
      pnpmBinPath,
      electronVersion,
      nodeBinDir: pnpmRuntime.nodeBinDir,
      nodeShimPath: pnpmRuntime.nodeShimPath,
      clearEnvironmentPath: pnpmRuntime.clearEnvironmentPath,
      dshBootstrapPath,
    }
    await healDesktopProfileModuleFallback(homeDir, prepared.profile)
    if (profilePreferences === undefined) {
      throw new Error(`${BIN_NAME}: active Profile preferences were not initialized`)
    }
    let currentProfilePreferences: DesktopProfilePreferences = profilePreferences
    let profilePreferencesWriteTail: Promise<void> = Promise.resolve()
    let profilePreferencesStopping = false
    const enqueueProfilePreferencesWrite = (
      update: (current: DesktopProfilePreferences) => DesktopProfilePreferences,
    ): Promise<DesktopProfilePreferencesStateV1> => {
      if (profilePreferencesStopping) {
        return Promise.reject(new Error(`${BIN_NAME}: Profile preferences are stopping`))
      }
      const write = profilePreferencesWriteTail.then(async () => {
        const next = update(currentProfilePreferences)
        const stored = await writeDesktopProfilePreferences(
          marketUserDataDir,
          prepared.profile.dir,
          next,
        )
        currentProfilePreferences = stored
        return stored
      })
      profilePreferencesWriteTail = write.then(() => undefined, () => undefined)
      return write
    }
    const flushProfilePreferencesWrites = async (): Promise<void> => {
      profilePreferencesStopping = true
      await profilePreferencesWriteTail
    }
    startupStage = 'host-boot'
    lifecycleRecorder.transitionStartupStage(startupStage)
    const releasePackageResolver = installProfilePackageResolver(prepared.bareModuleBaseUrl)
    const ctx = await boot(
      BIN_NAME,
      prepared.rootConfig,
      prepared.patches,
      async (hostCtx) => {
        // Keep Host imports and browser bundle discovery on the same public
        // profile-overlay resolver used by packaged Electron.
        hostCtx.loader.internal = undefined
        generation.bindHost(hostCtx)
        hostCtx.effect(
          () => async () => { await flushProfilePreferencesWrites() },
          'dsh-plugin-desktop: flush Profile preference writes',
        )
        hostCtx.effect(
          () => releasePnpmRuntime,
          'dsh-plugin-desktop: packaged pnpm runtime PATH',
        )
        if (dshRuntime !== undefined) {
          hostCtx.effect(
            () => releaseDshRuntime,
            'dsh-plugin-desktop: packaged dsh runtime PATH',
          )
        }
        hostCtx.effect(
          () => releasePackageResolver,
          'dsh-plugin-desktop: profile package resolution',
        )
        hostCtx.provide(DSH_LAUNCH_ENVIRONMENT_KEY, environment)
        hostCtx.provide('desktopBrowserAccess', browserAccess)
        hostCtx.provide('desktopLanHttps', lanHttps)
        hostCtx.provide('desktopRuntime', runtime)
        hostCtx.provide('desktopPnpmBootstrap', desktopPnpmBootstrap)
        await hostCtx.plugin(DesktopActionsService, {
          openTerminal: () => { runtime.openTerminal() },
          requestRestart: () => runtime.requestRestart(),
        })
        if (prepared.market.effective === 'community-market') {
          await hostCtx.plugin(DesktopPluginsService, {
            profileName: activeProfileName,
            homeDir,
            statePath: pluginManagementStatePath,
            installAnchor: desktopInstallAnchor(),
          })
        }
        if (logSink !== undefined) {
          fileExporter = new FileExporter(logSink)
          hostCtx.logger.exporter(fileExporter)
        }
        await hostCtx.plugin(DesktopProfileService, {
          current: {
            name: activeProfileName,
            dir: prepared.profile.dir,
          },
          create: name => createFreshDesktopProfile(name),
          list: () => listDesktopProfiles(homeDir),
          canDelete: name => canDeleteDesktopProfile({
            home: homeDir,
            selectionStatePath,
            currentProfileName: activeProfileName,
          }, name),
          delete: async name => {
            const profileDir = resolveProfileDir(name, homeDir)
            await deleteDesktopProfile({
              home: homeDir,
              selectionStatePath,
              currentProfileName: activeProfileName,
              clearDisabledState: () => clearDesktopProfilePluginState(pluginManagementStatePath, name),
              clearCheckpoint: async () => {
                clearDesktopProfileUsageHistory(releaseUserDataLocations, profileDir)
              },
            }, name)
            try {
              await clearDesktopProfilePreferences(marketUserDataDir, profileDir)
            } catch (cause) {
              hostCtx.logger.error(
                `${BIN_NAME}: deleted Profile left stale preference state: ${cause instanceof Error ? cause.message : String(cause)}`,
              )
            }
          },
          persistSelection: name => { selectDesktopProfile(selectionStatePath, homeDir, name) },
          requestRestart: () => runtime.requestRestart(),
        })
        let pendingSettingsRestart: ReturnType<typeof setImmediate> | undefined
        const scheduleSettingsRestart = (): void => {
          pendingSettingsRestart ??= setImmediate(() => {
            pendingSettingsRestart = undefined
            void runtime.requestRestart().catch((cause: unknown) => {
              hostCtx.logger.error(
                `${BIN_NAME}: failed to restart after Desktop setting change: ${cause instanceof Error ? cause.message : String(cause)}`,
              )
            })
          })
        }
        hostCtx.effect(() => () => {
          if (pendingSettingsRestart !== undefined) clearImmediate(pendingSettingsRestart)
          pendingSettingsRestart = undefined
        }, 'dsh-plugin-desktop: pending Desktop settings restart')
        const readMarket = () => desktopMarketSnapshotWithEffective(
          desktopProfileMarketSnapshot(currentProfilePreferences.market),
          prepared.market.effective,
        )
        hostCtx.provide('desktopSettingsController', new DesktopSettingsController({
          profiles: hostCtx.desktopProfiles,
          readMarket,
          readWeb: () => {
            const lan = lanHttps.snapshot()
            const lanOrigins = lan.state === 'ready' && lan.actualPort !== null
              ? desktopLanBrowserUrls(lan.actualPort, lan.addresses)
              : []
            return {
              localUrl: hostCtx.connection.authenticatedUrl(
                desktopLoopbackBrowserUrl(hostCtx.webServer.port),
              ),
              lanUrls: lanOrigins.map(url => hostCtx.connection.authenticatedUrl(url)),
              lanState: lan.state,
              lanError: lan.errorCode,
              lanCaFingerprint: lan.caFingerprint,
              lanCaUrls: lanOrigins.map((origin) => {
                return new URL(DESKTOP_LAN_HTTPS_CA_PATH, origin).href
              }),
            }
          },
          selectMarket: async provider => {
            await enqueueProfilePreferencesWrite(current => desktopProfilePreferencesFromSettings(
              current,
              current.notifications,
              provider,
            ))
            return desktopMarketSnapshotWithEffective(
              await selectDesktopMarketProvider(marketUserDataDir, provider),
              prepared.market.effective,
            )
          },
          scheduleRestart: scheduleSettingsRestart,
          scheduleRecoveryRestart: () => {
            void runtime.requestRecoveryRestart().catch((cause: unknown) => {
              hostCtx.logger.error(
                `${BIN_NAME}: failed to restart in recovery mode: ${cause instanceof Error ? cause.message : String(cause)}`,
              )
            })
          },
          openTerminal: () => { runtime.openTerminal() },
          reloadRenderer: () => { runtime.reloadRenderer() },
          toggleDeveloperTools: () => { runtime.toggleDeveloperTools() },
          exportDiagnostics: () => runtime.exportDiagnostics(),
          openProfileCreator: () => {
            runtime.openProfileCreateWindow({
              onSubmit: async name => {
                hostCtx.desktopProfiles.create(name)
                await hostCtx.desktopProfiles.select(name)
              },
            })
          },
        }))
        provideCmdline(hostCtx, {
          args: [
            '--port',
            String(prepared.port),
          ],
          exit: requestQuit,
        })
      },
      prepared.bareModuleBaseUrl,
    ).catch((cause: unknown) => {
      releasePackageResolver()
      throw cause
    })
    generation.bindHost(ctx)
    fileExporter?.setThreshold((ctx.settings.get(DESKTOP_SETTINGS_NAMESPACE) as DesktopSettings | undefined)?.logLevel ?? 'info')
    ctx.on('settings/updated', (namespace, next) => {
      if (namespace === DESKTOP_SETTINGS_NAMESPACE) {
        fileExporter?.setThreshold((next as DesktopSettings).logLevel)
      }
      if (namespace !== DESKTOP_SETTINGS_NAMESPACE
        && namespace !== DESKTOP_NOTIFICATIONS_SETTINGS_NAMESPACE) return
      const write = enqueueProfilePreferencesWrite(current => desktopProfilePreferencesFromSettings(
        namespace === DESKTOP_SETTINGS_NAMESPACE
          ? next as DesktopSettings
          : ctx.settings.get(DESKTOP_SETTINGS_NAMESPACE) as DesktopSettings,
        namespace === DESKTOP_NOTIFICATIONS_SETTINGS_NAMESPACE
          ? next as DesktopNotificationSettings
          : ctx.settings.get(DESKTOP_NOTIFICATIONS_SETTINGS_NAMESPACE) as DesktopNotificationSettings,
        current.market,
      ))
      void write.catch((cause: unknown) => {
        ctx.logger.error(
          `${BIN_NAME}: failed to capture active Profile settings: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      })
    })
    if (desktopWailsHostSidecarRequested()) {
      // Hybrid Wails shell: Cordis Host serves the UI; Electron skips BrowserWindow/Tray.
      // Prefer Wails loopback auth-proxy (injects x-dsh-desktop-renderer). Keep
      // ordinary loopback access as fallback when the proxy is unavailable
      // (Linux WebKitGTK cannot inject per-request headers natively).
      browserAccess.setOrdinaryBrowserEnabled(true)
      const hostUiUrl = ctx.connection.authenticatedUrl(
        desktopLoopbackBrowserUrl(ctx.webServer.port),
      )
      announceWailsHostReady(hostUiUrl)
      announceWailsHostAuthHeader(
        browserAccess.rendererHeader.name,
        browserAccess.rendererHeader.value,
      )
      try {
        announceWailsHostLanHttps(lanHttps.snapshot())
      } catch (cause) {
        electronLogger.error(
          `${BIN_NAME}: failed to announce LAN HTTPS snapshot: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      }
      electronLogger.error(
        `${BIN_NAME}: Wails Host sidecar ready at ${hostUiUrl} (Electron-light GUI skipped=${String(wailsElectronLight)})`,
      )
      const rendererReport = { status: 'healthy' as const }
      lifecycleRecorder.startRendererBoot()
      lifecycleRecorder.finishRendererBoot(rendererReport, 'renderer-failed')
      startupStage = 'health-commit'
      lifecycleRecorder.transitionStartupStage(startupStage)
      try {
        profileCheckpoint?.captureHealthy()
      } catch (cause) {
        electronLogger.error(
          `${BIN_NAME}: failed to checkpoint the healthy profile configuration: ${cause instanceof Error ? cause.message : String(cause)}`,
        )
      }
      lifecycleRecorder.completeStartup(startupStage, rendererReport)
      notifySkippedOptionalEntries(runtime, electronLogger, prepared.skippedOptionalEntries)
      notifyWindowsVolumeConcerns(runtime, electronLogger, windowsVolumeConcerns)
      if (sessionProjectionCacheRecovery !== undefined) {
        notifySessionProjectionCacheRecovery(runtime, electronLogger, sessionProjectionCacheRecovery)
      }
      return
    }
    startupStage = 'renderer-startup'
    lifecycleRecorder.transitionStartupStage(startupStage)
    lifecycleRecorder.startRendererBoot()
    const rendererBoot = runtime.beginRendererBootMonitoring({
      commitHealthy: async () => {
        lifecycleRecorder.finishRendererBoot({ status: 'healthy' }, 'renderer-failed')
        startupStage = 'health-commit'
        lifecycleRecorder.transitionStartupStage(startupStage)
        try {
          profileCheckpoint?.captureHealthy()
        } catch (cause) {
          electronLogger.error(
            `${BIN_NAME}: failed to checkpoint the healthy profile configuration: ${cause instanceof Error ? cause.message : String(cause)}`,
          )
        }
      },
    })
    const [, rendererVerdict] = await Promise.all([
      runtime.mountScheduled(),
      rendererBoot,
    ])
    const rendererReport = rendererVerdict.report
    if ('failureReason' in rendererVerdict) {
      throw new RendererStartupFailure(
        rendererVerdict.failureReason,
        rendererVerdict.report,
      )
    }
    lifecycleRecorder.completeStartup(startupStage, rendererReport)
    notifySkippedOptionalEntries(runtime, electronLogger, prepared.skippedOptionalEntries)
    notifyWindowsVolumeConcerns(runtime, electronLogger, windowsVolumeConcerns)
    if (safeModePaths !== undefined && DESKTOP_SAFE_MODE_DEFAULTS.settings.notifications.enabled) {
      notifyDesktopSafeModeActive(runtime, electronLogger)
    }
    if (sessionProjectionCacheRecovery !== undefined) {
      notifySessionProjectionCacheRecovery(runtime, electronLogger, sessionProjectionCacheRecovery)
    }
  } catch (cause) {
    runtime.stopRendererBootMonitoring()
    lifecycleRecorder.failRendererBootIfPending(lifecycleRendererFailureReason(runtime.rendererBootFailureReason))
    lifecycleRecorder.failStartup(startupStage, lifecycleStartupFailureReason(cause, runtime))
    electronLogger.errorCause(cause)
    let exitCode = 1
    const failureRoute = routeDesktopStartupFailure({
      appReady: false,
      stage: startupStage,
    })
    const recoveryActionsSafe = await generation.quiesceForRecovery()
    if (failureRoute === 'startup-recovery') {
      const detail = cause instanceof Error ? cause.message : String(cause)
      const recoveryResult = await openStartupRecoveryWindow(
        detail,
        recoveryActionsSafe ? startupRecoveryController : undefined,
      )
      if (recoveryResult === 'restart') {
        nativeExit.requestRelaunch()
        exitCode = 0
      } else if (recoveryResult === 'safe-mode') {
        nativeExit.requestRelaunch(desktopSafeModeRelaunchArguments())
        exitCode = 0
      }
    }
    startupRecoveryController?.dispose()
    await shutdown.request(exitCode)
  }
}

async function run(): Promise<void> {
  process.title = PRODUCT_NAME
  if (process.argv.includes('--export-diagnostics')) {
    try {
      const { exportDesktopDiagnostics } = await import('./diagnostic-export.ts')
      const path = await exportDesktopDiagnostics(resolveDesktopUserDataDirectory(), {
        appVersion: desktopProductVersion(),
      })
      process.stdout.write(`${path}\n`)
      process.exitCode = 0
    } catch (cause) {
      process.stderr.write(
        `dsh-plugin-desktop: failed to export diagnostics: ${cause instanceof Error ? cause.stack ?? cause.message : String(cause)}\n`,
      )
      process.exitCode = 1
    }
    return
  }
  await start()
}

/** Last-resort branch for launcher failures before the owned coordinator exists. */
async function handleFatalLauncherFailure(cause: unknown): Promise<void> {
  const detail = maskSecrets(cause instanceof Error ? cause.stack ?? cause.message : String(cause))
  process.stderr.write(`${BIN_NAME}: fatal Node Host failure: ${detail}\n`)
  try {
    announceWailsHostRecoveryRequired(detail)
  } catch {
    // ignore announce failures during fatal handling
  }
  process.exitCode = 1
}

void run().catch(cause => {
  void handleFatalLauncherFailure(cause)
})
