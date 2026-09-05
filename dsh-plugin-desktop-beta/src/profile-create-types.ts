/** Shared profile-create option types (no native shell imports). */

import type { DesktopLocale } from './runtime.ts'

export interface ProfileCreateWindowOptions {
  readonly locale: DesktopLocale
  readonly onSubmit: (name: string) => void | Promise<void>
  readonly onCancel?: () => void
}
