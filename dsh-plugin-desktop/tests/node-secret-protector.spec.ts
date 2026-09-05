import { mkdtempSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { describe, expect, it } from "vitest"
import { createLanHttpsCertificate } from "../src/lan-https-certificate.ts"
import {
  createNodeLanHttpsPrivateKeyProtector,
  desktopNodeSecretProtectorKeyPath,
} from "../src/node-secret-protector.ts"

describe("node-secret-protector", () => {
  it("roundtrips seal/open and persists a master key", async () => {
    const userData = mkdtempSync(join(tmpdir(), "dsh-nsp-"))
    const protector = createNodeLanHttpsPrivateKeyProtector(userData)
    expect(await protector.available()).toBe(true)
    const sealed = await protector.seal(Buffer.from("secret-ca-key", "utf8"))
    const opened = await protector.open(sealed)
    expect(Buffer.from(opened).toString("utf8")).toBe("secret-ca-key")
    expect(desktopNodeSecretProtectorKeyPath(userData)).toContain("lan-https")
  })

  it("works with createLanHttpsCertificate", async () => {
    const userData = mkdtempSync(join(tmpdir(), "dsh-nsp-cert-"))
    const protector = createNodeLanHttpsPrivateKeyProtector(userData)
    const first = await createLanHttpsCertificate(userData, ["127.0.0.1"], protector)
    const second = await createLanHttpsCertificate(userData, ["127.0.0.1"], protector)
    expect(first.caFingerprint).toBe(second.caFingerprint)
    expect(first.caCertificate).toContain("BEGIN CERTIFICATE")
  })
})
