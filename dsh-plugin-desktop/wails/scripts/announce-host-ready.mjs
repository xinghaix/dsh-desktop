#!/usr/bin/env node
/**
 * Tiny helper for hybrid Wails + Cordis Host boots.
 *
 * Usage:
 *   node announce-host-ready.mjs http://127.0.0.1:PORT/
 *
 * Effects:
 *   - prints `DSH_HOST_READY <url>` to stdout (consumed by Go HostSidecar)
 *   - if DSH_HOST_URL_FILE is set, writes the URL into that file
 */
import { writeFileSync } from 'node:fs'

const url = process.argv[2]
if (!url) {
  console.error('usage: announce-host-ready.mjs <host-ui-url>')
  process.exit(2)
}

process.stdout.write(`DSH_HOST_READY ${url}\n`)
const file = process.env.DSH_HOST_URL_FILE
if (file) {
  writeFileSync(file, `${url}\n`, 'utf8')
}
