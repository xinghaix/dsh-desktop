package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AuthProxy is a loopback reverse proxy that injects the Desktop renderer
// access header on every upstream request. Wails v3 beta does not expose
// pluggable webview request-header hooks:
//   - Windows WebView2: processRequest only sets UA / window-id internally
//   - macOS WKWebView: no public custom-header API in Wails v3 beta
//   - Linux WebKitGTK: cannot inject per-request headers
// The proxy is therefore the default production auth path on all platforms.
//
// Cordis Host authenticates via /?token=… → Set-Cookie → redirect /. The
// Wails webview does not reliably retain that cookie, so AuthProxy keeps a
// server-side CookieJar, bootstraps the token exchange itself, and injects
// Cookie on every upstream request.
type AuthProxy struct {
	mu         sync.Mutex
	listener   net.Listener
	server     *http.Server
	upstream   string
	headerName string
	headerVal  string
	proxyURL   string
	required   bool
	jar        http.CookieJar
	cookieN    int
}

func NewAuthProxy() *AuthProxy {
	// Production default: AuthProxy is required whenever a renderer token exists.
	return &AuthProxy{required: true}
}

// Status reports whether the auth proxy is listening.
func (p *AuthProxy) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	mode := "production-required"
	if !p.required {
		mode = "optional"
	}
	if p.proxyURL == "" {
		return fmt.Sprintf("auth-proxy=stopped mode=%s", mode)
	}
	return fmt.Sprintf("auth-proxy=listening mode=%s url=%s upstream=%s header=%s cookies=%d",
		mode, p.proxyURL, p.upstream, p.headerName, p.cookieN)
}

// ProxyURL returns the loopback URL the webview should load, if running.
func (p *AuthProxy) ProxyURL() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.proxyURL
}

// StartListening begins a loopback reverse proxy to upstreamURL that injects
// headerName/headerValue and jar cookies on every request (including WebSocket
// upgrades). When upstreamURL carries a token (or similar auth) query, the
// proxy performs the Host cookie exchange server-side and returns
// http://127.0.0.1:PORT/ so the webview does not depend on the token query.
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
	host := strings.ToLower(parsed.Hostname())
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return "", fmt.Errorf("auth-proxy upstream must be loopback (got %q)", parsed.Hostname())
	}
	origin := &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}

	bootstrapped := false
	if hasAuthQuery(parsed) {
		if err := bootstrapHostSession(parsed, headerName, headerValue, jar); err != nil {
			return "", fmt.Errorf("auth-proxy bootstrap: %w", err)
		}
		bootstrapped = true
	}

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
	cookieJar := jar
	proxy.Director = func(req *http.Request) {
		director(req)
		// Strip hop-by-hop headers that should not be forwarded.
		for _, h := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailers", "Transfer-Encoding", "Upgrade"} {
			if strings.EqualFold(h, "Upgrade") || strings.EqualFold(h, "Connection") {
				continue // allow websocket upgrades
			}
			req.Header.Del(h)
		}
		req.Header.Set(headerN, headerV)
		req.Header.Set("X-DSH-Auth-Proxy", "1")
		// Avoid leaking the proxy Host to upstream when Host expects loopback.
		req.Host = origin.Host
		injectJarCookies(req, cookieJar)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		ingestSetCookies(resp, cookieJar)
		// Primary auth is jar injection on the next hop; drop upstream
		// Set-Cookie so the webview does not see Domain/Path for a different port.
		resp.Header.Del("Set-Cookie")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		http.Error(w, "dsh auth-proxy: "+e.Error(), http.StatusBadGateway)
	}

	srv := &http.Server{
		Handler:           proxy,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	p.listener = ln
	p.server = srv
	p.upstream = origin.String()
	p.headerName = headerName
	p.headerVal = headerValue
	p.jar = jar
	p.cookieN = len(jar.Cookies(origin))
	addr := ln.Addr().(*net.TCPAddr)
	local := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", addr.Port),
		Path:   "/",
	}
	if !bootstrapped {
		local.Path = parsed.Path
		local.RawQuery = parsed.RawQuery
		local.Fragment = parsed.Fragment
		if local.Path == "" {
			local.Path = "/"
		}
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
	p.jar = nil
	p.cookieN = 0
	return err
}

// PlatformAuthCapability documents native webview header injection support.
func PlatformAuthCapability() string {
	return "native-header-injection=unavailable (Wails v3 beta); AuthProxy is the default production path on all platforms; " +
		"Windows WebView2 has internal WebResourceRequested but no public hook; " +
		"macOS WKWebView / Linux WebKitGTK cannot inject x-dsh-desktop-renderer per request; " +
		"AuthProxy binds 127.0.0.1 only, rejects non-loopback upstreams, and keeps a CookieJar for Host token→cookie auth"
}

func hasAuthQuery(u *url.URL) bool {
	if u == nil {
		return false
	}
	q := u.Query()
	return q.Get("token") != "" || q.Get("access_token") != "" || q.Get("auth") != ""
}

func bootstrapHostSession(upstream *url.URL, headerName, headerValue string, jar http.CookieJar) error {
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
	req, err := http.NewRequest(http.MethodGet, upstream.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set(headerName, headerValue)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("host session exchange returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func injectJarCookies(req *http.Request, jar http.CookieJar) {
	if req == nil || jar == nil || req.URL == nil {
		return
	}
	cookies := jar.Cookies(req.URL)
	if len(cookies) == 0 {
		return
	}
	req.Header.Del("Cookie")
	for _, c := range cookies {
		req.AddCookie(c)
	}
}

func ingestSetCookies(resp *http.Response, jar http.CookieJar) {
	if resp == nil || jar == nil || resp.Request == nil || resp.Request.URL == nil {
		return
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return
	}
	jar.SetCookies(resp.Request.URL, cookies)
}
