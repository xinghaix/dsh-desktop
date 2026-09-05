import { mkdtempSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { planHostSidecarSpawn, resolveHostLauncherMode } from '../src/host-launcher.ts'

describe('host-launcher', () => {
  it('defaults to node when host-main exists', () => {
    const hostMainPath = join(mkdtempSync(join(tmpdir(), 'dsh-hl-')), 'host-main.js')
    writeFileSync(hostMainPath, 'ok\n')
    expect(resolveHostLauncherMode({ environment: {}, hostMainPath }).mode).toBe('node')
    expect(resolveHostLauncherMode({ environment: { DSH_HOST_LAUNCHER: 'node' }, hostMainPath }).mode).toBe('node')
  })

  it('rejects removed launcher modes', () => {
    const hostMainPath = join(mkdtempSync(join(tmpdir(), 'dsh-hl-')), 'host-main.js')
    writeFileSync(hostMainPath, 'ok\n')
    expect(() => resolveHostLauncherMode({ environment: { DSH_HOST_LAUNCHER: "electron-as-node" }, hostMainPath })).toThrow(/removed/)
    expect(() => resolveHostLauncherMode({ environment: { DSH_HOST_LAUNCHER: "electron-main" }, hostMainPath })).toThrow(/removed/)
  })

  it('plans stock Node spawn', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'dsh-plan-'))
    const hostMainPath = join(dir, 'host-main.js')
    writeFileSync(hostMainPath, 'ok\n')
    const plan = await planHostSidecarSpawn({ hostMainPath, environment: {} })
    expect(plan.mode).toBe('node')
    expect(plan.execPath).toBe(process.execPath)
    expect(plan.argv[0]).toBe(hostMainPath)
    expect(plan.env.ELECTRON_RUN_AS_NODE).toBeUndefined()
  })
})
