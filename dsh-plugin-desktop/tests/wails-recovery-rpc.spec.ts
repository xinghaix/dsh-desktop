import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  DesktopStartupRecoveryControllerError,
  type DesktopStartupRecoverySnapshot,
} from '../src/startup-recovery-controller.ts'
import {
  announceWailsHostRecoveryRpc,
  DSH_HOST_RECOVERY_RPC_PREFIX,
  parseWailsHostRecoveryRpcAnnounce,
  startWailsRecoveryRpcServer,
  type WailsRecoveryRpcController,
} from '../src/wails-recovery-rpc.ts'

describe('Wails Recovery RPC', () => {
  const servers: Array<{ close: () => Promise<void> }> = []
  afterEach(async () => {
    for (const server of servers.splice(0)) await server.close().catch(() => {})
    vi.restoreAllMocks()
  })

  it('parses and announces recovery rpc lines', () => {
    const write = vi.spyOn(process.stdout, 'write').mockImplementation(() => true)
    announceWailsHostRecoveryRpc('http://127.0.0.1:9/', 'tok')
    expect(write).toHaveBeenCalledWith(`${DSH_HOST_RECOVERY_RPC_PREFIX}http://127.0.0.1:9/ token=tok\n`)
    expect(parseWailsHostRecoveryRpcAnnounce(`${DSH_HOST_RECOVERY_RPC_PREFIX}http://127.0.0.1:9/ token=tok`))
      .toEqual({ url: 'http://127.0.0.1:9/', token: 'tok' })
  })

  it('serves structural snapshot without a controller', async () => {
    const server = await startWailsRecoveryRpcServer({ detail: 'need-recovery', token: 'test-token' })
    servers.push(server)
    const health = await fetch(`${server.url}v1/health`, {
      headers: { authorization: `Bearer ${server.token}` },
    })
    expect(health.status).toBe(200)
    await expect(health.json()).resolves.toMatchObject({
      ok: true,
      detail: 'need-recovery',
      hasController: false,
    })
    const snapshot = await fetch(`${server.url}v1/snapshot`, {
      headers: { authorization: `Bearer ${server.token}` },
    })
    expect(snapshot.status).toBe(200)
    await expect(snapshot.json()).resolves.toEqual({
      profileName: '',
      bundles: [],
      checkpoints: [],
      controller: false,
    })
  })

  it('rejects missing bearer token', async () => {
    const server = await startWailsRecoveryRpcServer({ token: 'secret' })
    servers.push(server)
    const res = await fetch(`${server.url}v1/health`)
    expect(res.status).toBe(401)
  })

  it('round-trips checkpoint preview against a mock controller', async () => {
    const snapshot: DesktopStartupRecoverySnapshot = {
      profileName: 'default',
      bundles: [],
      checkpoints: [{ slotId: 'slot-1', status: 'empty' }],
    }
    const controller: WailsRecoveryRpcController = {
      async snapshot() {
        return snapshot
      },
      async previewCheckpointRestore(slotId) {
        if (slotId !== 'slot-1') {
          throw new DesktopStartupRecoveryControllerError('invalid-target', 'bad slot')
        }
        return {
          previewId: 'restore_abcdefghijklmnopqrstuvwxyz0123456789ABCD',
          slotId: 'slot-1',
          capturedAt: '2026-09-05T00:00:00.000Z',
          expiresAt: '2026-09-05T00:05:00.000Z',
        }
      },
      async executeCheckpointRestore() {
        throw new DesktopStartupRecoveryControllerError('state-unavailable', 'not in this test')
      },
      async previewUninstall() {
        throw new DesktopStartupRecoveryControllerError('invalid-target', 'none')
      },
      async executeUninstall() {
        throw new DesktopStartupRecoveryControllerError('invalid-target', 'none')
      },
    }
    const server = await startWailsRecoveryRpcServer({ controller, token: 'tok', detail: 'requested' })
    servers.push(server)
    const snapRes = await fetch(`${server.url}v1/snapshot`, {
      headers: { authorization: 'Bearer tok' },
    })
    expect(snapRes.status).toBe(200)
    await expect(snapRes.json()).resolves.toMatchObject({
      profileName: 'default',
      controller: true,
      checkpoints: [{ slotId: 'slot-1', status: 'empty' }],
    })
    const previewRes = await fetch(`${server.url}v1/checkpoint/preview`, {
      method: 'POST',
      headers: { authorization: 'Bearer tok', 'content-type': 'application/json' },
      body: JSON.stringify({ slotId: 'slot-1' }),
    })
    expect(previewRes.status).toBe(200)
    await expect(previewRes.json()).resolves.toMatchObject({
      slotId: 'slot-1',
      previewId: expect.stringMatching(/^restore_/),
    })
    const bad = await fetch(`${server.url}v1/checkpoint/preview`, {
      method: 'POST',
      headers: { authorization: 'Bearer tok', 'content-type': 'application/json' },
      body: JSON.stringify({ slotId: 'slot-9' }),
    })
    expect(bad.status).toBe(400)
    await expect(bad.json()).resolves.toMatchObject({
      error: { code: 'invalid-target' },
    })
  })

  it('resolves waitForComplete on /v1/complete', async () => {
    const server = await startWailsRecoveryRpcServer({ token: 'tok' })
    servers.push(server)
    const waiting = server.waitForComplete()
    const res = await fetch(`${server.url}v1/complete`, {
      method: 'POST',
      headers: { authorization: 'Bearer tok', 'content-type': 'application/json' },
      body: JSON.stringify({ action: 'restart' }),
    })
    expect(res.status).toBe(200)
    await expect(waiting).resolves.toBe('restart')
  })
})
