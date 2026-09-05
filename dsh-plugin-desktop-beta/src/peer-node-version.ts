/** Resolve Node version used for native module installs. */

export function resolveDesktopNodeVersion(
  _moduleUrl: string = import.meta.url,
  versions: NodeJS.ProcessVersions = process.versions,
): string {
  const live = versions.node
  if (typeof live === 'string' && live.length > 0) return live
  throw new Error('dsh-plugin-desktop: process.versions.node unavailable')
}

/** @deprecated use resolveDesktopNodeVersion */
export const resolveDesktopElectronVersion = resolveDesktopNodeVersion
