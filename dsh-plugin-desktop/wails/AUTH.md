# Wails auth / renderer header

Native per-request x-dsh-desktop-renderer hooks are unavailable on Wails v3 beta (Windows/macOS/Linux).
AuthProxy (loopback-only, timeouts, X-DSH-Auth-Proxy marker) is the default production path.
See authproxy.go and bridgeservice.go.
