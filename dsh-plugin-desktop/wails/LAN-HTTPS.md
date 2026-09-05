# LAN HTTPS in the Wails hybrid shell

TLS termination remains **Host-owned** (`DesktopLanHttpsRuntime` / `LanHttpsIngress` in
`dsh-plugin-desktop/src/`). The Wails shell does **not** terminate HTTPS and does not mint
the installation-local CA.

## Hybrid bridge

1. Host creates the installation-local CA + leaf certificate during Cordis boot (same as Electron).
2. When sidecar mode is active, Host announces `DSH_HOST_LAN_HTTPS` at ready time and again
   whenever the edge is enabled/disabled (`announceWailsHostLanHttps` in `src/wails-host-sidecar.ts`).
3. Announce fields (space-separated `key=value`):
   - `state` — runtime edge state (`ready` / disabled / error-ish Host values)
   - `port` — bound port or `null`
   - `addresses` — CSV of advertised IPv4 addresses, or `-`
   - `fingerprint` — CA fingerprint or `null`
   - `error` — error code or `null`
   - `urls` — CSV of HTTPS URLs when ready, or `-`
4. `HostSidecar` forwards matching stdout lines to
   `CapabilitiesService.IngestLanHttpsAnnounceLine`.
5. Operators inspect Tools → **LAN HTTPS Status** or Help → **Capabilities Status**.
   **LAN HTTPS Status** is multi-line (state/port/addresses/fingerprint/error/urls) for
   dialog readability. **Capabilities Status** keeps a compact `lan-https=announced …`
   one-liner. Awaiting state explains Host ownership until the first valid announce
   (no double `lan-https=` prefix).

Example Host stdout line:

```text
DSH_HOST_LAN_HTTPS state=ready port=8443 addresses=192.168.1.10 fingerprint=abcd error=null urls=-
```

## Smoke checks (hybrid)

1. run the primary hybrid shell entry
2. Confirm Host log contains DSH_HOST_READY and DSH_HOST_LAN_HTTPS when LAN exposure is enabled.
3. Tools menu LAN HTTPS Status shows lan-https=announced with the same fields.
4. Help menu Capabilities Status includes the same lan-https= line plus updates/crash/identity.

Until announce arrives, awaiting status is expected and not a shell failure.

## Private-key protection

| Host path | Protector |
| --- | --- |
| Electron main.ts | Electron safeStorage (OS keychain / DPAPI; Linux rejects basic_text) |
| Node host-main.ts | createNodeLanHttpsPrivateKeyProtector AES-GCM under userData/lan-https/ |

### Node protector threat model

- Protects against: other local OS users reading ca.json when directory modes are honored; casual offline inspection of sealedPrivateKey without the master key file.
- Does not protect against: same-UID malware; memory disclosure while Host is running; backups that include both ca.json and the master key.
- Stronger alternative: Electron safeStorage when running under Electron main.
- Sealed blobs use magic prefix DSHK1. Electron-sealed state cannot be opened by the Node protector; recreate the LAN HTTPS CA under Node Host if migrating.

## Still Host / product debt

- Enabling LAN exposure is driven by Desktop settings (networkExposure=lan) inside Cordis, not by a Wails-native toggle yet.
- CA download path /.well-known/dsh-desktop-ca.crt is served by the Host web carrier.
- Wails only mirrors announce state for operator visibility; binding and trust UX stay Host-owned.
