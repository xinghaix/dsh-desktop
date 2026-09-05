/** Shared Host recovery types (no native shell imports). */

export type RecoveryWindowResult = 'restart' | 'safe-mode' | 'quit'

export type { DesktopStartupFailureStage } from './recovery-copy.ts'

export interface DesktopStartupRecoveryProfile {
  readonly name: string
  readonly current: boolean
  readonly selectable: boolean
}

export interface DesktopStartupRecoveryProfileActions {
  readonly token: string
  readonly list: () => readonly DesktopStartupRecoveryProfile[]
  readonly switchProfile: (name: string, token: string) => void | Promise<void>
  readonly openCreator: () => void | Promise<void>
}

export interface DesktopStartupRecoveryConfigurationPaths {
  readonly settingsDocument: string
  readonly profilePatch: string
  readonly profileManifest: string
  readonly profileDirectory: string
}
