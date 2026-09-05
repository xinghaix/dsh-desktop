package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

// AuthProxy is a loopback reverse proxy that injects the Desktop renderer
// access header on every upstream request. Wails v3 beta does not expose
// pluggable webview request-header hooks:
//   - Windows WebView2: processRequest only sets UA / window-id internally
//   - macOS WKWebView: no public custom-header API in Wails v3 beta
//   - Linux WebKitGTK: cannot inject per-request headers
// The proxy is therefore the best available cross-platform auth path.
type AuthProxy struct {
	mu         sync.Mutex
	listener   net.Listener
	server     *http.Server
	upstream   string
	headerName string
	headerVal  string
	proxyURL   string
}

func NewAuthProxy() *AuthProxy {
	return &AuthProxy{}
}

// Status reports whether the auth proxy is listening.
func (p *AuthProxy) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proxyURL == "" {
		return "auth-proxy=stopped"
	}
	return fmt.Sprintf("auth-proxy=listening url=%s upstream=%s header=%s", p.proxyURL, p.upstream, p.headerName)
}

// ProxyURL returns the loopback URL the webview should load, if running.
func (p *AuthProxy) ProxyURL() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.proxyURL
}

// StartListening begins a loopback reverse proxy to upstreamURL that injects
// headerName/headerValue on every request (including WebSocket upgrades).
// Returns the local URL (preserving path/query from upstreamURL).
func (p *AuthProxy) StartListening(upstreamURL, headerName, headerValue string) (string, error) {
	upstreamURL = strings.TrimSpace(upstreamURL)
	headerName = strings.TrimSpace(strings.ToLower(headerName))
	headerValue = strings.TrimSpace(headerValue)
	if upstreamURL == "" {
		return "", fmt.Errorf("upstream URL is required")
	}
	if headerName == "" || headerValue == "" {
		return "", fmt.Errorf("renderer access header name and value are required")
	}
	parsed, err := url.Parse(upstreamURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported upstream scheme %q", parsed.Scheme)
	}
	origin := &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener != nil {
		_ = p.listener.Close()
		p.listener = nil
		p.server = nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	proxy := httputil.NewSingleHostReverseProxy(origin)
	director := proxy.Director
	headerN := headerName
	headerV := headerValue
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Header.Set(headerN, headerV)
		// Avoid leaking the proxy Host to upstream when Host expects loopback.
		req.Host = origin.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		http.Error(w, "dsh auth-proxy: "+e.Error(), http.StatusBadGateway)
	}

	srv := &http.Server{Handler: proxy}
	p.listener = ln
	p.server = srv
	p.upstream = origin.String()
	p.headerName = headerName
	p.headerVal = headerValue
	addr := ln.Addr().(*net.TCPAddr)
	local := &url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("127.0.0.1:%d", addr.Port),
		Path:     parsed.Path,
		RawQuery: parsed.RawQuery,
		Fragment: parsed.Fragment,
	}
	if local.Path == "" {
		local.Path = "/"
	}
	p.proxyURL = local.String()

	go func() {
		_ = srv.Serve(ln)
	}()
	return p.proxyURL, nil
}

// Stop shuts down the auth proxy listener.
func (p *AuthProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener == nil {
		return nil
	}
	err := p.listener.Close()
	p.listener = nil
	p.server = nil
	p.proxyURL = ""
	return err
}

// PlatformAuthCapability documents native webview header injection support.
func PlatformAuthCapability() string {
	return "native-header-injection=unavailable (Wails v3 beta); use AuthProxy on all platforms; " +
		"Windows WebView2 has internal WebResourceRequested but no public hook; " +
		"macOS WKWebView / Linux WebKitGTK cannot inject x-dsh-desktop-renderer per request"
}
