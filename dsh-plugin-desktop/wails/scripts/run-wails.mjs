import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const goBin = process.env.DSH_GO_BIN || '/home/box/sdk/go1.27.0/bin'
const env = {
  ...process.env,
  PATH: `${goBin}:/home/box/go/bin:${process.env.PATH || ''}`,
  GOPATH: process.env.GOPATH || '/home/box/go',
}

const mode = process.argv[2] || 'run'
const rest = process.argv.slice(3)

function run(cmd, args, opts = {}) {
  const result = spawnSync(cmd, args, { cwd: root, env, stdio: 'inherit', ...opts })
  if (result.error) throw result.error
  if (result.status) process.exit(result.status)
}

switch (mode) {
  case 'build':
    run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    console.log(`built ${path.join(root, 'bin/dsh-wails-shell')}`)
    break
  case 'build:wails3': {
    const has = spawnSync('wails3', ['version'], { cwd: root, env, encoding: 'utf8' })
    if (has.status === 0) run('wails3', ['build', ...rest])
    else {
      console.error('wails3 not on PATH; falling back to go build')
      run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    }
    break
  }
  case 'package': {
    const has = spawnSync('wails3', ['version'], { cwd: root, env, encoding: 'utf8' })
    if (has.status !== 0) {
      console.error('wails3 not on PATH')
      process.exit(1)
    }
    const pkg = spawnSync('wails3', ['package', ...rest], { cwd: root, env, stdio: 'inherit' })
    if (pkg.status) {
      console.error('wails3 package failed or unsupported on this host; try build first')
      run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
      process.exit(pkg.status || 1)
    }
    break
  }
  case 'run':
  case 'dev':
  case 'start':
    run('go', ['build', '-o', 'bin/dsh-wails-shell', '.'])
    run(path.join(root, 'bin/dsh-wails-shell'), rest)
    break
  default:
    console.error('usage: run-wails.mjs {build|build:wails3|package|run|dev|start} [args...]')
    process.exit(2)
}
