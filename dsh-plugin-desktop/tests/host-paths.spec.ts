import { describe, expect, it } from "vitest"
import {
  resolveDesktopAppDataDirectory,
  resolveDesktopHomeDirectory,
  resolveDesktopUserDataDirectory,
} from "../src/host-paths.ts"

describe("host-paths", () => {
  it("resolves userData without Electron", () => {
    expect(resolveDesktopUserDataDirectory("win32", { APPDATA: "C:\\Users\\Example\\AppData\\Roaming" }, "ignored"))
      .toBe("C:\\Users\\Example\\AppData\\Roaming\\DSH Desktop")
    expect(resolveDesktopUserDataDirectory("darwin", {}, "/Users/example"))
      .toBe("/Users/example/Library/Application Support/DSH Desktop")
    expect(resolveDesktopUserDataDirectory("linux", {}, "/home/example"))
      .toBe("/home/example/.config/DSH Desktop")
  })

  it("resolves appData parent and home", () => {
    expect(resolveDesktopAppDataDirectory("linux", { XDG_CONFIG_HOME: "/xdg" }, "/home/x")).toBe("/xdg")
    expect(resolveDesktopHomeDirectory({ HOME: "/custom" }, "/fallback")).toBe("/custom")
  })
})
