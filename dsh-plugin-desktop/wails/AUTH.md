# Wails auth / renderer header

## Platform limits (Wails v3.0.0-beta.16)

| Platform | Webview | Per-request `x-dsh-desktop-renderer` |
| --- | --- | --- |
| Windows | WebView2 | Internal `WebResourceRequested` exists but Wails does not expose a public hook for custom headers |
| macOS | WKWebView | No public custom request-header API in Wails v3 beta |
| Linux | WebKitGTK | Cannot inject per-request headers |

## Best available path (hybrid)

1. Host sidecar announces `DSH_HOST_AUTH_HEADER x-dsh-desktop-renderer <token>`.
2. `BridgeService` stores the token and starts a loopback **AuthProxy** (`authproxy.go`).
3. The proxy injects the header on every upstream request (including upgrades).
4. `ShellService.StartHostSidecar` prefers the proxied URL for `LoadHostURL`.
5. Host may still enable ordinary loopback browser access as a fallback when the proxy is unavailable.

Native header injection remains blocked until Wails exposes platform hooks; do not assume Mac is "solved" by WKWebView alone.
