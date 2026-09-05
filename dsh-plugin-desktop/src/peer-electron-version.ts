/** Read peer Electron version without an Electron process. */

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

export function resolveDesktopElectronVersion(
  moduleUrl: string = import.meta.url,
  versions: NodeJS.ProcessVersions = process.versions,
): string {
  const live = versions.electron
  if (typeof live === 'string' && live.length > 0) return live
  const manifestPath = fileURLToPath(new URL('../package.json', moduleUrl))
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8')) as {
    peerDependencies?: { electron?: unknown }
    devDependencies?: { electron?: unknown }
  }
  const peer = manifest.peerDependencies?.electron
  if (typeof peer === 'string' && peer.length > 0) return peer
  const dev = manifest.devDependencies?.electron
  if (typeof dev === 'string' && dev.length > 0) return dev
  throw new Error('dsh-plugin-desktop: package.json has no electron peer/devDependency version')
}
