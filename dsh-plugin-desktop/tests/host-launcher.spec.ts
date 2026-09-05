import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { describe, expect, it } from "vitest"
import {
  electronNativeAddonsLikelyRequired,
  planHostSidecarSpawn,
  resolveHostLauncherMode,
} from "../src/host-launcher.ts"

describe("host-launcher", () => {
  it("honors DSH_HOST_LAUNCHER overrides", () => {
    const hostMainPath = join(mkdtempSync(join(tmpdir(), "dsh-hl-")), "host-main.js")
    writeFileSync(hostMainPath, "ok\n")
    expect(resolveHostLauncherMode({
      environment: { DSH_HOST_LAUNCHER: "electron-as-node" },
      hostMainPath,
      scanRoots: [],
    }).mode).toBe("electron-as-node")
    expect(resolveHostLauncherMode({
      environment: { DSH_HOST_LAUNCHER: "node" },
      hostMainPath,
      scanRoots: [],
    }).mode).toBe("node")
  })

  it("detects profile node_modules as Electron ABI likely", () => {
    const root = mkdtempSync(join(tmpdir(), "dsh-prof-"))
    const profile = join(root, "demo")
    mkdirSync(join(profile, "node_modules"), { recursive: true })
    expect(electronNativeAddonsLikelyRequired([root])).toBe(true)
    expect(electronNativeAddonsLikelyRequired([join(root, "missing")])).toBe(false)
  })

  it("plans ELECTRON_RUN_AS_NODE spawn when forced", async () => {
    const dir = mkdtempSync(join(tmpdir(), "dsh-plan-"))
    const hostMainPath = join(dir, "host-main.js")
    const electronMainPath = join(dir, "main.js")
    const electronPath = join(dir, "electron-bin")
    writeFileSync(hostMainPath, "ok\n")
    writeFileSync(electronMainPath, "ok\n")
    writeFileSync(electronPath, "#!/bin/sh\n")
    const plan = await planHostSidecarSpawn({
      hostMainPath,
      electronMainPath,
      environment: {
        DSH_HOST_LAUNCHER: "electron-as-node",
        ELECTRON_PATH: electronPath,
      },
      scanRoots: [],
    })
    expect(plan.mode).toBe("electron-as-node")
    expect(plan.execPath).toBe(electronPath)
    expect(plan.env.ELECTRON_RUN_AS_NODE).toBe("1")
    expect(plan.argv[0]).toBe(hostMainPath)
  })
})
