#!/usr/bin/env node
import { cpSync, existsSync, mkdirSync, rmSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"
const wailsRoot = join(dirname(fileURLToPath(import.meta.url)), "..")
const pluginRoot = join(wailsRoot, "..")
const src = join(pluginRoot, "lib", "native-ui")
const dest = join(wailsRoot, "frontend", "dist", "aux", "native-ui")
if (!existsSync(src)) { console.error("lib/native-ui missing — build plugin first (non-fatal)"); process.exit(0) }
rmSync(dest, { recursive: true, force: true })
mkdirSync(dirname(dest), { recursive: true })
cpSync(src, dest, { recursive: true })
console.log("synced native-ui ->", dest)
