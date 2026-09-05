#!/usr/bin/env node
import { cpSync, existsSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const wailsRoot = join(dirname(fileURLToPath(import.meta.url)), "..")
const pluginRoot = join(wailsRoot, "..")
const src = join(pluginRoot, "lib", "native-ui")
const dest = join(wailsRoot, "frontend", "dist", "shell-ui", "native-ui")

// Electron ships a lock-down CSP (default-src 'none'; connect-src 'none'). That
// combination blanks React in Wails WebKitGTK on the wails:// asset scheme and
// also blocks ExecJS scheme-bridge injection. Rewrite on sync for the shell copy.
const WAILS_CSP =
  `<meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'self'; form-action 'self'" />`

if (!existsSync(src)) {
  console.error("lib/native-ui missing — build plugin first (non-fatal)")
  process.exit(0)
}
rmSync(dest, { recursive: true, force: true })
mkdirSync(dirname(dest), { recursive: true })
cpSync(src, dest, { recursive: true })

for (const name of readdirSync(dest)) {
  if (!name.endsWith(".html")) continue
  const path = join(dest, name)
  const before = readFileSync(path, "utf8")
  const after = before.replace(/<meta http-equiv="Content-Security-Policy"[^>]*>/, WAILS_CSP)
  if (after !== before) writeFileSync(path, after)
}
console.log("synced native-ui ->", dest)
