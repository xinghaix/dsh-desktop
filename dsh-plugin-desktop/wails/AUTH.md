# Wails auth / renderer header

Native per-request x-dsh-desktop-renderer hooks are unavailable on Wails v3 beta
(Windows/macOS/Linux). AuthProxy (loopback-only, timeouts, X-DSH-Auth-Proxy marker,
CookieJar for Host token to session, Origin/Referer rewrite to upstream) is the
default production path. See authproxy.go and bridgeservice.go.

## Dev / hybrid

1. Host announces renderer access header on sidecar stdout.
2. BridgeService.StartAuthProxy binds 127.0.0.1 and rejects non-loopback upstreams.
3. Shell loads the AuthProxy URL into the webview when a token exists.

## Packaged artifacts

| Artifact | AuthProxy expectation |
| --- | --- |
| bin/dsh-wails-shell (smoke / package fallback) | Same production AuthProxy; Host sidecar required |
| wails3 package AppImage/deb (when tooling exists) | Must ship Go AuthProxy; no Electron session.webRequest |
| electron-builder installers (current product CI) | Electron session headers; AuthProxy unused |

Until the release flip, shipped downloads remain electron-builder. Preview Wails
packages must keep AuthProxy required:true and Origin rewrite (CSRF otherwise fails).

Smoke: node scripts/wails-smoke.mjs then Help -> Auth / Renderer Header.
