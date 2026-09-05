import { describe, expect, it } from "vitest"
import { resolveDesktopElectronVersion } from "../src/peer-electron-version.ts"

describe("peer-electron-version", () => {
  it("prefers live process.versions.electron", () => {
    expect(resolveDesktopElectronVersion(import.meta.url, { electron: "99.0.0" } as NodeJS.ProcessVersions)).toBe("99.0.0")
  })

  it("falls back to package peerDependency", () => {
    const version = resolveDesktopElectronVersion(import.meta.url, {} as NodeJS.ProcessVersions)
    expect(version).toMatch(/^\d+\.\d+\.\d+/)
  })
})
