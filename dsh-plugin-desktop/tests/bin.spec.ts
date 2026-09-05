import { existsSync, mkdtempSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { describe, expect, it, vi } from 'vitest'
import AdmZip from 'adm-zip'
import {
  defaultDesktopUserDataDirectory,
  DESKTOP_CLI_HELP,
  parseDesktopCli,
  runDesktopCli,
} from '../src/bin.ts'

vi.mock('electron', () => ({ default: undefined }))

describe('desktop npm launcher', () => {
  it('launches with no arguments', () => {
    expect(parseDesktopCli([])).toBe('launch')
  })

  it.each([
    ['--help', 'help'],
    ['-h', 'help'],
    ['--version', 'version'],
    ['-V', 'version'],
    ['--export-diagnostics', 'export-diagnostics'],
  ] as const)('parses %s', (argument, action) => {
    expect(parseDesktopCli([argument])).toBe(action)
  })

  it('rejects arguments that belong to the profile app', () => {
    expect(() => parseDesktopCli(['--port', '3000'])).toThrow('unknown arguments')
  })

  it('names the installed product and selected profile behavior', () => {
    expect(DESKTOP_CLI_HELP).toContain('DSH Desktop')
    expect(DESKTOP_CLI_HELP).toContain('selected Web-capable profile')
    expect(DESKTOP_CLI_HELP).toContain('--export-diagnostics')
  })

  it('resolves the packaged Desktop user-data directory without Electron', () => {
    expect(defaultDesktopUserDataDirectory('win32', { APPDATA: 'C:\\Users\\Example\\AppData\\Roaming' }, 'ignored'))
      .toBe('C:\\Users\\Example\\AppData\\Roaming\\DSH Desktop')
    expect(defaultDesktopUserDataDirectory('darwin', {}, '/Users/example'))
      .toBe('/Users/example/Library/Application Support/DSH Desktop')
  })

  it('exports diagnostics without launching Electron', async () => {
    const userDataDir = mkdtempSync(join(tmpdir(), 'dsh-cli-diagnostics-'))
    const write = vi.spyOn(process.stdout, 'write').mockImplementation(() => true)
    try {
      await expect(runDesktopCli(['--export-diagnostics'], { userDataDir })).resolves.toBe(0)
      const output = String(write.mock.calls.at(-1)?.[0]).trim()
      expect(existsSync(output)).toBe(true)
      expect(new AdmZip(output).readAsText('system-info.txt')).toMatch(/desktop-version: \d/u)
    } finally {
      write.mockRestore()
    }
  })
  it('defaults empty argv to Node Host launch path', async () => {
    // runDesktopCli([]) plans Node host-main; missing lib/host-main.js rejects.
    await expect(runDesktopCli([])).rejects.toThrow(/host-main|rebuild/i)
  })
})

