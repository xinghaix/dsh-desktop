/** Node Host private-key protector for LAN HTTPS (AES-256-GCM file-backed master key). */

import { createCipheriv, createDecipheriv, randomBytes } from "node:crypto"
import { chmodSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import type { DesktopLanHttpsPrivateKeyProtector } from "./lan-https-certificate.ts"

const PRIVATE_DIRECTORY_MODE = 0o700
const PRIVATE_FILE_MODE = 0o600
const MASTER_KEY_BYTES = 32
const IV_BYTES = 12
const MAGIC = Buffer.from("DSHK1", "utf8")
const MASTER_KEY_NAME = ".protector-master-key"

export const NODE_SECRET_PROTECTOR_THREAT_MODEL = Object.freeze({
  algorithm: "AES-256-GCM",
  keyStorage: "userData/lan-https/.protector-master-key (mode 0600)",
  protectsAgainst: [
    "other local OS users reading ca.json when directory modes are honored",
    "casual offline inspection of sealedPrivateKey without the master key file",
  ],
  doesNotProtectAgainst: [
    "same-UID malware or an attacker who can read the master key file",
    "memory disclosure of plaintext keys while Host is running",
    "compromised backups that include both ca.json and the master key",
  ],
  strongerAlternative: "Electron safeStorage (OS keychain / DPAPI) when running under Electron main",
})

function ensurePrivateDir(path: string): void {
  mkdirSync(path, { recursive: true, mode: PRIVATE_DIRECTORY_MODE })
  try { chmodSync(path, PRIVATE_DIRECTORY_MODE) } catch { /* best-effort on win32 */ }
}

function loadOrCreateMasterKey(keyPath: string): Buffer {
  if (existsSync(keyPath)) {
    const key = readFileSync(keyPath)
    if (key.length !== MASTER_KEY_BYTES) {
      throw new Error(`dsh-plugin-desktop: LAN HTTPS protector master key must be ${String(MASTER_KEY_BYTES)} bytes`)
    }
    return key
  }
  ensurePrivateDir(dirname(keyPath))
  const key = randomBytes(MASTER_KEY_BYTES)
  writeFileSync(keyPath, key, { mode: PRIVATE_FILE_MODE })
  try { chmodSync(keyPath, PRIVATE_FILE_MODE) } catch { /* best-effort */ }
  return key
}

export function desktopNodeSecretProtectorKeyPath(userDataDir: string): string {
  return join(userDataDir, "lan-https", MASTER_KEY_NAME)
}

/** Create a DesktopLanHttpsPrivateKeyProtector backed by a local AES master key. */
export function createNodeLanHttpsPrivateKeyProtector(
  userDataDir: string,
): DesktopLanHttpsPrivateKeyProtector {
  const keyPath = desktopNodeSecretProtectorKeyPath(userDataDir)
  return {
    available: () => {
      try {
        loadOrCreateMasterKey(keyPath)
        return true
      } catch {
        return false
      }
    },
    seal: (plaintext: Uint8Array) => {
      const key = loadOrCreateMasterKey(keyPath)
      const iv = randomBytes(IV_BYTES)
      const cipher = createCipheriv("aes-256-gcm", key, iv)
      const ciphertext = Buffer.concat([cipher.update(Buffer.from(plaintext)), cipher.final()])
      const tag = cipher.getAuthTag()
      return Buffer.concat([MAGIC, iv, tag, ciphertext])
    },
    open: (sealed: Uint8Array) => {
      const buf = Buffer.from(sealed)
      if (buf.length < MAGIC.length + IV_BYTES + 16) {
        throw new Error("dsh-plugin-desktop: sealed LAN HTTPS key is truncated")
      }
      if (!buf.subarray(0, MAGIC.length).equals(MAGIC)) {
        throw new Error(
          "dsh-plugin-desktop: sealed LAN HTTPS key is not Node protector format (recreate CA under Node Host or use Electron safeStorage)",
        )
      }
      const key = loadOrCreateMasterKey(keyPath)
      const iv = buf.subarray(MAGIC.length, MAGIC.length + IV_BYTES)
      const tag = buf.subarray(MAGIC.length + IV_BYTES, MAGIC.length + IV_BYTES + 16)
      const ciphertext = buf.subarray(MAGIC.length + IV_BYTES + 16)
      const decipher = createDecipheriv("aes-256-gcm", key, iv)
      decipher.setAuthTag(tag)
      return Buffer.concat([decipher.update(ciphertext), decipher.final()])
    },
  }
}
