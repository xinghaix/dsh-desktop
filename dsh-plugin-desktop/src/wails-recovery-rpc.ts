/** Loopback HTTP Recovery RPC for Wails Host-sidecar mode. */

import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http'
import { randomBytes } from 'node:crypto'
import { writeFileSync } from 'node:fs'
import {
  DesktopStartupRecoveryControllerError,
  type DesktopStartupRecoveryCheckpointPreview,
  type DesktopStartupRecoveryCheckpointResult,
  type DesktopStartupRecoverySnapshot,
  type DesktopStartupRecoveryUninstallPreview,
  type DesktopStartupRecoveryUninstallResult,
} from './startup-recovery-controller.ts'

/** Stdout line prefix for the Recovery RPC base URL + bearer token. */
export const DSH_HOST_RECOVERY_RPC_PREFIX = 'DSH_HOST_RECOVERY_RPC '

export type WailsRecoveryCompleteAction = 'restart' | 'safe-mode' | 'quit'

/** Minimal controller surface the Recovery RPC server requires. */
export interface WailsRecoveryRpcController {
  snapshot(): Promise<DesktopStartupRecoverySnapshot>
  previewCheckpointRestore(slotId: string): Promise<DesktopStartupRecoveryCheckpointPreview>
  executeCheckpointRestore(previewId: string): Promise<DesktopStartupRecoveryCheckpointResult>
  previewUninstall(bundleId: string): Promise<DesktopStartupRecoveryUninstallPreview>
  executeUninstall(previewId: string): Promise<DesktopStartupRecoveryUninstallResult>
  openCheckpoint?(slotId: string): Promise<void>
}

export interface WailsRecoveryRpcServer {
  readonly url: string
  readonly token: string
  readonly detail: string
  waitForComplete(): Promise<WailsRecoveryCompleteAction | 'unavailable'>
  close(): Promise<void>
}

export interface WailsRecoveryQuiesceResult {
  readonly ok: boolean
  readonly detail: string
}

export interface StartWailsRecoveryRpcServerOptions {
  readonly controller?: WailsRecoveryRpcController
  readonly detail?: string
  /** Injected listen port (tests); default ephemeral. */
  readonly port?: number
  readonly token?: string
  /**
   * Export the Electron/Host diagnostic archive zip (same format as --export-diagnostics).
   * Optional so structural servers and tests can omit it.
   */
  readonly exportDiagnostics?: () => Promise<string>
  /**
   * Best-effort generation quiesce before mutations / complete.
   * Real API is DesktopStartupGeneration.quiesceForRecovery() (Host fiber dispose + timeout).
   * There is no separate drain-generations / wait-for-idle / cancel-in-flight Host surface.
   */
  readonly quiesce?: () => Promise<WailsRecoveryQuiesceResult>
}

interface JsonErrorBody {
  readonly error: {
    readonly code: string
    readonly message: string
    readonly operationStage?: string
  }
}

/**
 * Announce Recovery RPC base URL + bearer token for the Go shell.
 * Format: `DSH_HOST_RECOVERY_RPC http://127.0.0.1:PORT/ token=<token>`
 */
export function announceWailsHostRecoveryRpc(
  url: string,
  token: string,
  environment: NodeJS.ProcessEnv = process.env,
): void {
  const trimmedUrl = url.trim()
  const trimmedToken = token.trim()
  if (trimmedUrl.length === 0 || trimmedToken.length === 0) {
    throw new Error('dsh-plugin-desktop: Wails Recovery RPC requires url and token')
  }
  process.stdout.write(`${DSH_HOST_RECOVERY_RPC_PREFIX}${trimmedUrl} token=${trimmedToken}\n`)
  const file = environment.DSH_HOST_RECOVERY_RPC_FILE
  if (file !== undefined && file.trim().length > 0) {
    writeFileSync(file.trim(), `${trimmedUrl} token=${trimmedToken}\n`, 'utf8')
  }
}

/** Parse a Recovery RPC announce line into base URL + token. */
export function parseWailsHostRecoveryRpcAnnounce(line: string): { url: string; token: string } | undefined {
  const trimmed = line.trim()
  if (!trimmed.startsWith(DSH_HOST_RECOVERY_RPC_PREFIX)) return undefined
  const rest = trimmed.slice(DSH_HOST_RECOVERY_RPC_PREFIX.length).trim()
  const match = /^(\S+)\s+token=(\S+)\s*$/u.exec(rest)
  if (match === null) return undefined
  return { url: match[1]!, token: match[2]! }
}

/**
 * Start a loopback-only Recovery RPC server backed by an optional controller.
 * Without a controller, health/snapshot still respond structurally (503 on mutations).
 */
export async function startWailsRecoveryRpcServer(
  options: StartWailsRecoveryRpcServerOptions = {},
): Promise<WailsRecoveryRpcServer> {
  const token = options.token ?? randomBytes(24).toString('base64url')
  const detail = (options.detail ?? 'recovery-required').trim() || 'recovery-required'
  const controller = options.controller

  let completeResolve: ((action: WailsRecoveryCompleteAction | 'unavailable') => void) | undefined
  const completePromise = new Promise<WailsRecoveryCompleteAction | 'unavailable'>(resolve => {
    completeResolve = resolve
  })
  let settled = false
  const settle = (action: WailsRecoveryCompleteAction | 'unavailable'): void => {
    if (settled) return
    settled = true
    completeResolve?.(action)
  }

  const exportDiagnostics = options.exportDiagnostics
  const quiesce = options.quiesce

  const server: Server = createServer((req, res) => {
    void handleRequest(req, res, {
      token,
      detail,
      controller,
      exportDiagnostics,
      quiesce,
      onComplete: action => {
        settle(action)
      },
    })
  })

  const url = await new Promise<string>((resolve, reject) => {
    server.once('error', reject)
    server.listen(options.port ?? 0, '127.0.0.1', () => {
      const address = server.address()
      if (address === null || typeof address === 'string') {
        reject(new Error('dsh-plugin-desktop: Recovery RPC failed to bind loopback port'))
        return
      }
      resolve(`http://127.0.0.1:${String(address.port)}/`)
    })
  })

  return {
    url,
    token,
    detail,
    waitForComplete: async () => await completePromise,
    close: async () => {
      settle('unavailable')
      await new Promise<void>((resolve, reject) => {
        server.close(error => {
          if (error) reject(error)
          else resolve()
        })
      })
    },
  }
}

interface RequestContext {
  readonly token: string
  readonly detail: string
  readonly controller: WailsRecoveryRpcController | undefined
  readonly exportDiagnostics: (() => Promise<string>) | undefined
  readonly quiesce: (() => Promise<WailsRecoveryQuiesceResult>) | undefined
  readonly onComplete: (action: WailsRecoveryCompleteAction) => void
}

async function handleRequest(
  req: IncomingMessage,
  res: ServerResponse,
  ctx: RequestContext,
): Promise<void> {
  try {
    if (!authorize(req, ctx.token)) {
      sendJson(res, 401, { error: { code: 'unauthorized', message: 'Recovery RPC requires bearer token' } })
      return
    }
    const path = (req.url ?? '/').split('?')[0] ?? '/'
    const method = (req.method ?? 'GET').toUpperCase()

    if (method === 'GET' && (path === '/v1/health' || path === '/health')) {
      sendJson(res, 200, {
        ok: true,
        detail: ctx.detail,
        hasController: ctx.controller !== undefined,
      })
      return
    }

    if (method === 'GET' && (path === '/v1/snapshot' || path === '/snapshot')) {
      if (ctx.controller === undefined) {
        // Structural empty snapshot so UI / clients can round-trip without a live controller.
        sendJson(res, 200, {
          profileName: '',
          bundles: [],
          checkpoints: [],
          controller: false,
        })
        return
      }
      const snapshot = await ctx.controller.snapshot()
      sendJson(res, 200, { ...snapshot, controller: true })
      return
    }

    if (method === 'POST' && path === '/v1/complete') {
      const body = await readJsonBody(req) as { action?: string }
      const action = body.action
      if (action !== 'restart' && action !== 'safe-mode' && action !== 'quit') {
        sendJson(res, 400, { error: { code: 'invalid-target', message: 'action must be restart|safe-mode|quit' } })
        return
      }
      const quiesce = await runQuiesce(ctx)
      ctx.onComplete(action)
      sendJson(res, 200, { ok: true, action, quiesce })
      return
    }

    if (method === 'POST' && path === '/v1/quiesce') {
      const quiesce = await runQuiesce(ctx)
      sendJson(res, 200, quiesce)
      return
    }

    if (method === 'POST' && path === '/v1/diagnostics/export') {
      if (ctx.exportDiagnostics === undefined) {
        sendJson(res, 501, {
          error: {
            code: 'state-unavailable',
            message: 'Diagnostic archive export is not wired for this Recovery RPC session',
          },
        })
        return
      }
      const archivePath = await ctx.exportDiagnostics()
      sendJson(res, 200, { ok: true, path: archivePath })
      return
    }

    if (method === 'POST' && path === '/v1/checkpoint/preview') {
      const body = await readJsonBody(req) as { slotId?: string }
      const slotId = typeof body.slotId === 'string' ? body.slotId : ''
      const result = await requireController(ctx).previewCheckpointRestore(slotId)
      sendJson(res, 200, result)
      return
    }

    if (method === 'POST' && path === '/v1/checkpoint/execute') {
      const body = await readJsonBody(req) as { previewId?: string }
      const previewId = typeof body.previewId === 'string' ? body.previewId : ''
      const result = await requireController(ctx).executeCheckpointRestore(previewId)
      sendJson(res, 200, result)
      return
    }

    if (method === 'POST' && path === '/v1/checkpoint/open') {
      const body = await readJsonBody(req) as { slotId?: string }
      const slotId = typeof body.slotId === 'string' ? body.slotId : ''
      const open = requireController(ctx).openCheckpoint
      if (open === undefined) {
        sendJson(res, 501, { error: { code: 'state-unavailable', message: 'openCheckpoint is not available' } })
        return
      }
      await open(slotId)
      sendJson(res, 200, { ok: true, slotId })
      return
    }

    if (method === 'POST' && path === '/v1/uninstall/preview') {
      const body = await readJsonBody(req) as { bundleId?: string }
      const bundleId = typeof body.bundleId === 'string' ? body.bundleId : ''
      const result = await requireController(ctx).previewUninstall(bundleId)
      sendJson(res, 200, result)
      return
    }

    if (method === 'POST' && path === '/v1/uninstall/execute') {
      const body = await readJsonBody(req) as { previewId?: string }
      const previewId = typeof body.previewId === 'string' ? body.previewId : ''
      const result = await requireController(ctx).executeUninstall(previewId)
      sendJson(res, 200, result)
      return
    }

    sendJson(res, 404, { error: { code: 'not-found', message: `unknown Recovery RPC path ${path}` } })
  } catch (cause) {
    sendControllerError(res, cause)
  }
}

async function runQuiesce(ctx: RequestContext): Promise<WailsRecoveryQuiesceResult> {
  if (ctx.quiesce === undefined) {
    return {
      ok: true,
      detail: 'quiesce=skipped (no Host generation callback; recovery keep-alive has no live Cordis Host fiber to dispose)',
    }
  }
  try {
    return await ctx.quiesce()
  } catch (cause) {
    const message = cause instanceof Error ? cause.message : String(cause)
    return { ok: false, detail: `quiesce=failed: ${message}` }
  }
}

function requireController(ctx: RequestContext): WailsRecoveryRpcController {
  if (ctx.controller === undefined) {
    throw new DesktopStartupRecoveryControllerError(
      'state-unavailable',
      'Desktop recovery controller is not available for this Host generation.',
    )
  }
  return ctx.controller
}

function authorize(req: IncomingMessage, token: string): boolean {
  const header = req.headers.authorization
  if (typeof header !== 'string') return false
  const match = /^Bearer\s+(\S+)\s*$/iu.exec(header)
  return match !== null && match[1] === token
}

async function readJsonBody(req: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = []
  for await (const chunk of req) {
    chunks.push(typeof chunk === 'string' ? Buffer.from(chunk) : chunk)
  }
  if (chunks.length === 0) return {}
  const raw = Buffer.concat(chunks).toString('utf8').trim()
  if (raw.length === 0) return {}
  return JSON.parse(raw) as unknown
}

function sendJson(res: ServerResponse, status: number, body: unknown): void {
  const payload = JSON.stringify(body)
  res.writeHead(status, {
    'content-type': 'application/json; charset=utf-8',
    'content-length': Buffer.byteLength(payload),
    'cache-control': 'no-store',
  })
  res.end(payload)
}

function sendControllerError(res: ServerResponse, cause: unknown): void {
  if (cause instanceof DesktopStartupRecoveryControllerError) {
    const body: JsonErrorBody = {
      error: {
        code: cause.code,
        message: cause.message,
        ...(cause.operationStage === undefined ? {} : { operationStage: cause.operationStage }),
      },
    }
    const status = cause.code === 'invalid-target' || cause.code === 'preview-expired'
      ? 400
      : cause.code === 'generation-changed' || cause.code === 'immutable-target'
        ? 409
        : cause.code === 'operation-in-progress'
          ? 409
          : cause.code === 'state-unavailable'
            ? 503
            : 500
    sendJson(res, status, body)
    return
  }
  if (cause instanceof SyntaxError) {
    sendJson(res, 400, { error: { code: 'invalid-target', message: 'Request body must be JSON' } })
    return
  }
  const message = cause instanceof Error ? cause.message : String(cause)
  sendJson(res, 500, { error: { code: 'operation-failed', message } })
}
