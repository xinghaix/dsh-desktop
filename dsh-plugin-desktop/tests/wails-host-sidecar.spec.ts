import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  DESKTOP_WAILS_HOST_SIDECAR_ARGUMENT,
  DSH_HOST_READY_PREFIX,
  DSH_WAILS_HOST_SIDECAR_ENV,
  announceWailsHostReady,
  desktopWailsHostSidecarRequested,
} from '../src/wails-host-sidecar.ts'

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
})
