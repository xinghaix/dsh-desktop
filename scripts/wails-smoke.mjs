#!/usr/bin/env node
/** Non-workflow Wails smoke (go test + go build via run-wails.mjs). */
import { spawnSync } from "node:child_process"
import { existsSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"
import os from "node:os"
const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..")
const runWails = join(repoRoot, "dsh-plugin-desktop", "wails", "scripts", "run-wails.mjs")
if (!existsSync(runWails)) { console.error("missing", runWails); process.exit(1) }
const env = { ...process.env }
const candidates = [process.env.DSH_GO_BIN, "/home/box/sdk/go1.27.0/bin", join(os.homedir(), "sdk", "go1.27.0", "bin"), join(os.homedir(), "go", "bin")].filter(Boolean)
for (const dir of candidates) {
  if (existsSync(join(dir, "go")) || existsSync(join(dir, "go.exe"))) {
    env.PATH = dir + (env.PATH ? ":" + env.PATH : "")
    env.DSH_GO_BIN = dir
    break
  }
}
env.GOPATH = env.GOPATH || join(os.homedir(), "go")
const result = spawnSync(process.execPath, [runWails, "smoke"], { cwd: join(repoRoot, "dsh-plugin-desktop"), env, stdio: "inherit" })
process.exit(result.status ?? 1)
