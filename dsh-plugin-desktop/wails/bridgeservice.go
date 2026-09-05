package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const desktopRendererAccessHeader = "x-dsh-desktop-renderer"
const hostAuthHeaderPrefix = "DSH_HOST_AUTH_HEADER "

// BridgeService replaces Electron preload/contextBridge + session auth for the
// hybrid Wails shell. Host UI still loads via authenticated loopback URL;
// renderer request-header injection is not available on Linux webkitgtk in
// Wails v3 beta, so sidecar mode enables ordinary-browser Host access and this
// service exposes native bindings the control UI / future Host bridge can call.
type BridgeService struct {
	mu         sync.Mutex
	app        *application.App
	shell      *ShellService
	proxy      *AuthProxy
	authHeader string
	authValue  string
	authURL    string
	lastAuth   string
}

func NewBridgeService(shell *ShellService) *BridgeService {
	return &BridgeService{shell: shell, proxy: NewAuthProxy()}
}

func (b *BridgeService) attach(app *application.App) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.app = app
}

// BridgeStatus reports auth-bridge readiness for the control UI.
// Never includes secret header values — only header name presence and proxy diagnostics.
func (b *BridgeService) BridgeStatus() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	header := "(none)"
	if b.authHeader != "" {
		header = b.authHeader
	}
	hasToken := "no"
	if b.authValue != "" {
		hasToken = "yes"
	}
	auth := b.lastAuth
	if auth == "" {
		auth = "(idle)"
	}
	proxyStatus := "auth-proxy=stopped"
	if b.proxy != nil {
		proxyStatus = b.proxy.Status()
	}
	return fmt.Sprintf("bridge=ready header=%s hasToken=%s lastAuth=%s %s", header, hasToken, auth, proxyStatus)
}

// SetRendererAccessHeader stores the Host generation's renderer capability.
// name should be x-dsh-desktop-renderer; value is the ephemeral token.
func (b *BridgeService) SetRendererAccessHeader(name, value string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	value = strings.TrimSpace(value)
	if name == "" || value == "" {
		return fmt.Errorf("renderer access header name and value are required")
	}
	if name != desktopRendererAccessHeader {
		return fmt.Errorf("unsupported renderer access header %q", name)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.authHeader = name
	b.authValue = value
	return nil
}

// RendererAccessHeaderName returns the configured header name, if any.
func (b *BridgeService) RendererAccessHeaderName() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.authHeader
}

// HasRendererAccessHeader reports whether a token is configured (not the value).
func (b *BridgeService) HasRendererAccessHeader() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.authHeader != "" && b.authValue != ""
}

// AuthenticateHostSession performs the Electron session.fetch equivalent:
// GET authenticationUrl with the renderer access header, accepting Set-Cookie.
// On success, navigates the main window to uiURL (usually the authenticated UI).
// Note: webview cookie import from the Go jar is not available on all platforms;
// the hybrid path therefore also relies on Host ordinary-browser access + loading
// the authenticated URL directly in the webview.
func (b *BridgeService) AuthenticateHostSession(authenticationURL, uiURL string) (string, error) {
	authenticationURL = strings.TrimSpace(authenticationURL)
	uiURL = strings.TrimSpace(uiURL)
	if authenticationURL == "" {
		return "", fmt.Errorf("authenticationURL is required")
	}
	if uiURL == "" {
		uiURL = authenticationURL
	}
	b.mu.Lock()
	headerName := b.authHeader
	headerValue := b.authValue
	b.mu.Unlock()

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if headerName != "" && headerValue != "" {
				req.Header.Set(headerName, headerValue)
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodGet, authenticationURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cache-Control", "no-store")
	if headerName != "" && headerValue != "" {
		req.Header.Set(headerName, headerValue)
	}
	resp, err := client.Do(req)
	if err != nil {
		b.setLastAuth("error:" + err.Error())
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("browser authentication failed with HTTP %d", resp.StatusCode)
		b.setLastAuth(msg)
		return "", fmt.Errorf("%s", msg)
	}
	b.mu.Lock()
	b.authURL = authenticationURL
	b.mu.Unlock()
	b.setLastAuth(fmt.Sprintf("ok cookies=%d ui=%s", len(jar.Cookies(mustURL(authenticationURL))), uiURL))

	if b.shell != nil {
		if err := b.shell.LoadHostURL(uiURL); err != nil {
			return uiURL, err
		}
	}
	return uiURL, nil
}

// GetPathForFile is the preload bridge stand-in. Electron used webUtils.getPathForFile;
// Wails/webkitgtk does not expose OS paths for drag File objects on this box, so
// callers receive a clear error instead of a fake path.
func (b *BridgeService) GetPathForFile(_ string) (string, error) {
	return "", fmt.Errorf("GetPathForFile is unavailable in the Wails hybrid shell on this platform; use ShellService.OpenDirectoryDialog / OpenFileDialog instead")
}

// OpenExternal opens a URL with the OS default handler (replaces shell.openExternal).
func (b *BridgeService) OpenExternal(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	switch parsed.Scheme {
	case "http", "https", "mailto":
	default:
		return fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	b.mu.Lock()
	app := b.app
	b.mu.Unlock()
	if app == nil {
		return fmt.Errorf("application is not attached")
	}
	return app.Browser.OpenURL(rawURL)
}

// DesktopFilePathBridgeKey returns the preload main-world key for Host compatibility docs.
func (b *BridgeService) DesktopFilePathBridgeKey() string {
	return "__DSH_DESKTOP_FILE_PATH__"
}

// IngestHostAuthHeaderLine parses "DSH_HOST_AUTH_HEADER <name> <value>" from Host stdout.
func (b *BridgeService) IngestHostAuthHeaderLine(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, hostAuthHeaderPrefix) {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, hostAuthHeaderPrefix))
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) != 2 {
		return false
	}
	_ = b.SetRendererAccessHeader(parts[0], parts[1])
	return true
}

// LoadAuthMetaFile reads optional JSON/plain meta written beside DSH_HOST_URL_FILE.
// Supported plain format:
//
//	url=http://127.0.0.1:PORT/...
//	header=x-dsh-desktop-renderer TOKEN
func (b *BridgeService) LoadAuthMetaFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "header=") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "header="))
			parts := strings.SplitN(rest, " ", 2)
			if len(parts) == 2 {
				_ = b.SetRendererAccessHeader(parts[0], parts[1])
			}
		}
	}
	return nil
}

func (b *BridgeService) setLastAuth(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastAuth = msg
}


// StartAuthProxy starts the loopback renderer-header proxy for upstreamURL.
func (b *BridgeService) StartAuthProxy(upstreamURL string) (string, error) {
	b.mu.Lock()
	headerName := b.authHeader
	headerValue := b.authValue
	proxy := b.proxy
	b.mu.Unlock()
	if proxy == nil {
		return "", fmt.Errorf("auth proxy is not configured")
	}
	if headerName == "" || headerValue == "" {
		return "", fmt.Errorf("renderer access header is not configured; wait for Host auth announce")
	}
	return proxy.StartListening(upstreamURL, headerName, headerValue)
}

// AuthProxyURL returns the current proxy URL, if any.
func (b *BridgeService) AuthProxyURL() string {
	b.mu.Lock()
	proxy := b.proxy
	b.mu.Unlock()
	if proxy == nil {
		return ""
	}
	return proxy.ProxyURL()
}

// AuthProxyStatus returns proxy diagnostics.
func (b *BridgeService) AuthProxyStatus() string {
	b.mu.Lock()
	proxy := b.proxy
	b.mu.Unlock()
	if proxy == nil {
		return "auth-proxy=(nil)"
	}
	return proxy.Status()
}

// PlatformAuthNotes returns Mac/Linux/Windows header-injection capability notes.
func (b *BridgeService) PlatformAuthNotes() string {
	return PlatformAuthCapability()
}

// PreferProxiedHostURL returns a proxy URL when the renderer header is known,
// otherwise returns upstream unchanged (ordinary-browser fallback).
func (b *BridgeService) PreferProxiedHostURL(upstreamURL string) (string, error) {
	upstreamURL = strings.TrimSpace(upstreamURL)
	if upstreamURL == "" {
		return "", fmt.Errorf("upstream URL is required")
	}
	b.mu.Lock()
	ready := b.authHeader != "" && b.authValue != ""
	b.mu.Unlock()
	if !ready {
		return upstreamURL, nil
	}
	proxied, err := b.StartAuthProxy(upstreamURL)
	if err != nil {
		return upstreamURL, err
	}
	return proxied, nil
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		return &url.URL{}
	}
	return u
}
