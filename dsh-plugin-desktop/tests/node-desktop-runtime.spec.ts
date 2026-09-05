import { mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { afterEach, describe, expect, it, vi } from "vitest"
import { NodeDesktopRuntime } from "../src/node-desktop-runtime.ts"
import type { DesktopShellSpec } from "../src/runtime.ts"
import { createDesktopBrowserAccess } from "../src/desktop-browser-access.ts"

describe("NodeDesktopRuntime", () => {
  const dirs: string[] = []
  afterEach(() => {
    for (const dir of dirs.splice(0)) rmSync(dir, { recursive: true, force: true })
  })

  it("schedules without mounting Electron windows", async () => {
    const dir = mkdtempSync(join(tmpdir(), "dsh-node-runtime-"))
    dirs.push(dir)
    const restart = vi.fn(async () => {})
    const runtime = new NodeDesktopRuntime(restart, undefined, dir)
    const dispose = runtime.schedule({
      mode: "compatibility",
      macosMaterial: "off",
      windowsMaterial: "off",
      width: 800,
      height: 600,
      minWidth: 400,
      minHeight: 300,
      material: "off",
      url: "http://127.0.0.1:9/",
      authenticationUrl: "http://127.0.0.1:9/?t=1",
      rendererAccessHeader: createDesktopBrowserAccess(true).rendererHeader,
      productName: "DSH Desktop",
      windowTitle: "t",
      iconPath: "/tmp/icon.png",
      trayIcons: { templatePath: "/tmp/t.png", bluePath: "/tmp/b.png" },
      readLocalePreference: () => undefined,
      readThemeSource: () => "system",
      requestQuit: () => {},
      requestModeChange: async () => {},
    } as DesktopShellSpec)
    expect(runtime.scheduledSpec).toBeDefined()
    await expect(runtime.mountScheduled()).resolves.toBeUndefined()
    await dispose()
    expect(runtime.scheduledSpec).toBeUndefined()
    runtime.registerTrayItem({
      group: "tools",
      order: 0,
      label: () => "x",
      invoke: () => {},
    }).dispose()
    await runtime.requestRestart()
    expect(restart).toHaveBeenCalled()
  })
})
