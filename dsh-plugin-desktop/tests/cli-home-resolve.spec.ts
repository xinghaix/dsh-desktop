import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { describe, expect, it, vi } from 'vitest'
import { resolveDesktopCliEntry, requireDesktopCliEntry } from '../src/cli-home-resolve.ts'
import { runDesktopDshCli } from '../src/desktop-cli.ts'

function makeCliHome(root: string) {
  const pkg = join(root, 'node_modules', '@deepseek-ai', 'dsh')
  mkdirSync(join(pkg, 'lib'), { recursive: true })
  writeFileSync(join(pkg, 'package.json'), JSON.stringify({ name: '@deepseek-ai/dsh', type: 'module' }))
  const entry = join(pkg, 'lib', 'bin.js')
  writeFileSync(entry, '')
  return entry
}

describe('home-first dsh CLI resolution', () => {
  it('uses ~/.dsh node_modules entry when present', () => {
    const home = mkdtempSync(join(tmpdir(), 'dsh-cli-home-'))
    try {
      const dot = join(home, '.dsh')
      const entry = makeCliHome(dot)
      const report = resolveDesktopCliEntry({ environment: {}, homeDirectory: home, bundledEntryPath: '/nope/bundled.js' })
      expect(report.hit?.path).toBe(entry)
      expect(report.hit?.reason).toContain('.dsh')
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  it('errors friendly when home CLI is missing (no silent bundled)', () => {
    const home = mkdtempSync(join(tmpdir(), 'dsh-cli-miss-'))
    try {
      const report = resolveDesktopCliEntry({ environment: {}, homeDirectory: home, bundledEntryPath: '/tmp/fake-bundled-bin.js' })
      expect(report.hit).toBeUndefined()
      expect(report.message).toContain('will not silently use the bundled')
      expect(report.message).toContain('Checked paths')
      expect(() => requireDesktopCliEntry({ environment: {}, homeDirectory: home })).toThrow(/will not silently use the bundled/)
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  it('allows bundled only when opt-in flag is set', () => {
    const home = mkdtempSync(join(tmpdir(), 'dsh-cli-bundled-'))
    const bundled = join(home, 'bundled-bin.js')
    try {
      writeFileSync(bundled, '')
      const off = resolveDesktopCliEntry({ environment: {}, homeDirectory: home, bundledEntryPath: bundled })
      expect(off.hit).toBeUndefined()
      const on = resolveDesktopCliEntry({ environment: { DSH_CLI_ALLOW_BUNDLED: '1' }, homeDirectory: home, bundledEntryPath: bundled })
      expect(on.hit?.path).toBe(bundled)
      expect(on.hit?.reason).toBe('DSH_CLI_ALLOW_BUNDLED')
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  it('prefers DSH_CLI_BIN over home', () => {
    const home = mkdtempSync(join(tmpdir(), 'dsh-cli-bin-'))
    try {
      makeCliHome(join(home, '.dsh'))
      const explicit = join(home, 'explicit-bin.js')
      writeFileSync(explicit, '')
      const report = resolveDesktopCliEntry({ environment: { DSH_CLI_BIN: explicit }, homeDirectory: home })
      expect(report.hit?.path).toBe(explicit)
      expect(report.hit?.reason).toBe('DSH_CLI_BIN')
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  it('ignores Desktop Host home lib/bin.js that is not @deepseek-ai/dsh', () => {
    const home = mkdtempSync(join(tmpdir(), 'dsh-cli-hosthome-'))
    try {
      const dot = join(home, '.dsh')
      mkdirSync(join(dot, 'lib'), { recursive: true })
      writeFileSync(join(dot, 'package.json'), JSON.stringify({ name: 'dsh-plugin-desktop' }))
      writeFileSync(join(dot, 'lib', 'bin.js'), '')
      const report = resolveDesktopCliEntry({ environment: {}, homeDirectory: home })
      expect(report.hit).toBeUndefined()
    } finally {
      rmSync(home, { recursive: true, force: true })
    }
  })

  it('runDesktopDshCli loads home entry and skips silent bundled', async () => {
    const home = mkdtempSync(join(tmpdir(), 'dsh-cli-run-'))
    const prevHome = process.env.HOME
    try {
      const entry = makeCliHome(join(home, '.dsh'))
      process.env.HOME = home
      const load = vi.fn(async (url: string) => {
        expect(url).toContain('node_modules')
        expect(url).toContain('bin.js')
      })
      const argv = ['node', 'desktop-cli.js', '--version']
      await runDesktopDshCli({ HOME: home }, load, argv)
      expect(load).toHaveBeenCalledOnce()
      expect(decodeURIComponent(String(load.mock.calls[0]?.[0]))).toContain(entry)
    } finally {
      if (prevHome === undefined) delete process.env.HOME
      else process.env.HOME = prevHome
      rmSync(home, { recursive: true, force: true })
    }
  })
})
