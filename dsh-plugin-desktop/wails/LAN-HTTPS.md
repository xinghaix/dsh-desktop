# LAN HTTPS in the Wails hybrid shell

TLS termination remains **Host-owned** (`DesktopLanHttpsRuntime` / `LanHttpsIngress` in `dsh-plugin-desktop/src/`). The Wails shell does not terminate HTTPS.

## Hybrid bridge

1. Host creates the installation-local CA + leaf certificate during Cordis boot (same as Electron).
2. When sidecar mode is active, Host announces `DSH_HOST_LAN_HTTPS …` at ready time and again whenever the edge is enabled/disabled.
3. Announce fields: `state`, `port`, `addresses`, `fingerprint`, `error`, `urls` (HTTPS URLs when ready).
4. `CapabilitiesService.IngestLanHttpsAnnounceLine` stores the latest line for the Tools → LAN HTTPS Status UI.

## Still Electron / Host debt

- Certificate private-key protection uses Electron `safeStorage` on some platforms.
- Enabling LAN exposure is driven by Desktop settings (`networkExposure=lan`) inside Cordis, not by a Wails-native toggle yet.
- CA download path `/.well-known/dsh-desktop-ca.crt` is served by the Host web carrier.
