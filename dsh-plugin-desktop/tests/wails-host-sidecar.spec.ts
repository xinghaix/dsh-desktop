import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT,
  DSH_HOST_AUTH_HEADER_PREFIX,
  DSH_HOST_LAN_HTTPS_PREFIX,
  DSH_HOST_READY_PREFIX,
  DSH_HOST_RECOVERY_REQUIRED_PREFIX,
  DSH_WAILS_HOST_SIDECAR_ENV,
  announceWailsHostAuthHeader,
  announceWailsHostLanHttps,
  announceWailsHostReady,
  announceWailsHostRecoveryRequired,
  desktopWailsHostSidecarRequested,
  desktopWailsSkipElectronGui,
} from '../src/wails-host-sidecar.ts'
import { desktopDefaultRelaunchArguments } from '../src/relaunch-arguments.ts'

describe('Wails Host sidecar helpers', () => {
  const dirs: string[] = []
  afterEach(() => {
    for (const dir of dirs.splice(0)) rmSync(dir, { recursive: true, force: true })
    vi.restoreAllMocks()
  })

  it('detects env and exact argv marker only', () => {
    expect(desktopWailsHostSidecarRequested(['node', 'main.js'], {})).toBe(false)
    expect(desktopWailsHostSidecarRequested(['node', 'main.js', DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT], {})).toBe(true)
    expect(desktopWailsHostSidecarRequested(['node', 'main.js'], { [DSH_WAILS_HOST_SIDECAR_ENV]: '1' })).toBe(true)
    expect(desktopWailsHostSidecarRequested(['node', `main.js`], { [DSH_WAILS_HOST_SIDECAR_ENV]: '0' })).toBe(false)
    expect(desktopWailsSkipElectronGui(['node', 'main.js', DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT], {})).toBe(true)
  })

  it('preserves sidecar marker across default relaunch argv filtering', () => {
    const relaunched = desktopDefaultRelaunchArguments([
      'electron',
      'main.js',
      DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT,
      '--dsh-desktop-recovery',
    ])
    expect(relaunched).toContain(DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT)
    expect(relaunched).not.toContain('--dsh-desktop-recovery')
  })

  it('announces stdout and optional URL file', () => {
    const dir = mkdtempSync(join(tmpdir(), 'dsh-wails-announce-'))
    dirs.push(dir)
    const file = join(dir, 'url.txt')
    const write = vi.spyOn(process.stdout, 'write').mockImplementation(() => true)
    announceWailsHostReady('http://127.0.0.1:43120/?token=abc', { DSH_HOST_URL_FILE: file })
    expect(write).toHaveBeenCalledWith(`${DSH_HOST_READY_PREFIX}http://127.0.0.1:43120/?token=abc\n`)
    expect(readFileSync(file, 'utf8')).toBe('http://127.0.0.1:43120/?token=abc\n')
  })

  it('announces auth header, recovery, and LAN HTTPS lines', () => {
    const write = vi.spyOn(process.stdout, 'write').mockImplementation(() => true)
    announceWailsHostAuthHeader('x-dsh-desktop-renderer', 'token')
    announceWailsHostRecoveryRequired('need-recovery')
    announceWailsHostLanHttps({
      state: 'inactive',
      actualPort: null,
      addresses: ['192.168.1.2'],
      caFingerprint: null,
      errorCode: null,
    })
    expect(write).toHaveBeenCalledWith(`${DSH_HOST_AUTH_HEADER_PREFIX}x-dsh-desktop-renderer token\n`)
    expect(write).toHaveBeenCalledWith(`${DSH_HOST_RECOVERY_REQUIRED_PREFIX}need-recovery\n`)
    expect(write).toHaveBeenCalledWith(
      `${DSH_HOST_LAN_HTTPS_PREFIX}state=inactive port=null addresses=192.168.1.2 fingerprint=null error=null urls=-\n`,
    )
  })
})
