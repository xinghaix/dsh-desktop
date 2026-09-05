import { spawnSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import os from 'node:os'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

/**
 * Prefer an explicit Go toolchain, then PATH `go`, then known local SDK layouts.
 * Avoid hardcoding cloud-box-only paths so Mac/local verify works.
 */
function resolveGoBinDir() {
  if (process.env.DSH_GO_BIN && process.env.DSH_GO_BIN.trim()) {
    return process.env.DSH_GO_BIN.trim()
  }
  const which = spawnSync('which', ['go'], { encoding: 'utf8' })
  if (which.status === 0) {
    const goPath = which.stdout.trim()
    if (goPath) return path.dirname(goPath)
  }
  const home = os.homedir()
  const candidates = [
    path.join(home, 'sdk', 'go1.27.0', 'bin'),
    path.join(home, 'go', 'bin'),
    '/home/box/sdk/go1.27.0/bin',
    '/usr/local/go/bin',
  ]
  for (const dir of candidates) {
    if (existsSync(path.join(dir, 'go')) || existsSync(path.join(dir, 'go.exe'))) {
      return dir
    }
  }
  return ''
}

const goBin = resolveGoBinDir()
const pathParts = []
if (goBin) pathParts.push(goBin)
if (process.env.GOPATH) pathParts.push(path.join(process.env.GOPATH, 'bin'))
else if (existsSync(path.join(os.homedir(), 'go', 'bin'))) {
  pathParts.push(path.join(os.homedir(), 'go', 'bin'))
}
pathParts.push(process.env.PATH || '')

const env = {
  ...process.env,
  PATH: pathParts.filter(Boolean).join(path.delimiter),
  GOPATH: process.env.GOPATH || path.join(os.homedir(), 'go'),
}

const mode = process.argv[2] || 'run'
const rest = process.argv.slice(3)

function run(cmd, args, opts = {}) {
  const result = spawnSync(cmd, args, { cwd: root, env, stdio: 'inherit', ...opts })
  if (result.error) throw result.error
  if (result.status) process.exit(result.status)
}

function ensureGo() {
  const probe = spawnSync('go', ['version'], { cwd: root, env, encoding: 'utf8' })
  if (probe.status !== 0) {
    console.error('go not found on PATH. Set DSH_GO_BIN to the directory containing the go binary, or install Go 1.27+.')
    process.exit(1)
  }
}

switch (mode) {
  case 'build':
    ensureGo()
    run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    console.log(`built ${path.join(root, 'bin/dsh-wails-shell')}`)
    break
  case 'build:wails3': {
    ensureGo()
    const has = spawnSync('wails3', ['version'], { cwd: root, env, encoding: 'utf8' })
    if (has.status === 0) run('wails3', ['build', ...rest])
    else {
      console.error('wails3 not on PATH; falling back to go build')
      run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    }
    break
  }
  case 'package': {
    ensureGo()
    const has = spawnSync('wails3', ['version'], { cwd: root, env, encoding: 'utf8' })
    if (has.status !== 0) {
      console.error('wails3 not on PATH; producing go-build artifact as package fallback')
      run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
      console.error('package:wails: wrote bin/dsh-wails-shell (installer packaging requires wails3 package on a supported host)')
      process.exit(0)
    }
    const pkg = spawnSync('wails3', ['package', ...rest], { cwd: root, env, stdio: 'inherit' })
    if (pkg.status) {
      console.error('wails3 package failed or unsupported on this host; falling back to go build artifact')
      run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
      console.error('package:wails: wrote bin/dsh-wails-shell')
      process.exit(pkg.status || 1)
    }
    break
  }
  case 'smoke':
    ensureGo()
    run('go', ['test', './...'])
    run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    console.log('wails smoke: go test + go build OK')
    break
  case 'run':
  case 'dev':
  case 'start':
    ensureGo()
    run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    run(path.join(root, 'bin/dsh-wails-shell'), rest)
    break
  default:
    console.error('usage: run-wails.mjs {build|build:wails3|package|smoke|run|dev|start} [args...]')
    process.exit(2)
}
