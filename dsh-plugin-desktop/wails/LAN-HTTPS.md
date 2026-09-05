# LAN HTTPS in the Wails hybrid shell

TLS termination remains **Host-owned** (`DesktopLanHttpsRuntime` / `LanHttpsIngress` in `dsh-plugin-desktop/src/`). The Wails shell does not terminate HTTPS.

## Hybrid bridge

1. Host creates the installation-local CA + leaf certificate during Cordis boot (same as Electron).
2. When sidecar mode is active, Host announces `DSH_HOST_LAN_HTTPS …` at ready time and again whenever the edge is enabled/disabled.
3. Announce fields: `state`, `port`, `addresses`, `fingerprint`, `error`, `urls` (HTTPS URLs when ready).
4. `CapabilitiesService.IngestLanHttpsAnnounceLine` stores the latest line for the Tools → LAN HTTPS Status UI.

## Private-key protection

| Host path | Protector |
| --- | --- |
| Electron `main.ts` | Electron `safeStorage` (OS keychain / DPAPI; Linux rejects `basic_text`) |
| Node `host-main.ts` | `createNodeLanHttpsPrivateKeyProtector` — AES-256-GCM with master key at `userData/lan-https/.protector-master-key` (mode 0600) |

### Node protector threat model

- **Protects against:** other local OS users reading `ca.json` when directory modes are honored; casual offline inspection of `sealedPrivateKey` without the master key file.
- **Does not protect against:** same-UID malware; memory disclosure while Host is running; backups that include both `ca.json` and the master key.
- **Stronger alternative:** Electron `safeStorage` when running under Electron main.
- Sealed blobs use magic prefix `DSHK1`. Electron-sealed state cannot be opened by the Node protector — recreate the LAN HTTPS CA under Node Host if migrating.

## Still Host / product debt

- Enabling LAN exposure is driven by Desktop settings (`networkExposure=lan`) inside Cordis, not by a Wails-native toggle yet.
- CA download path `/.well-known/dsh-desktop-ca.crt` is served by the Host web carrier.
