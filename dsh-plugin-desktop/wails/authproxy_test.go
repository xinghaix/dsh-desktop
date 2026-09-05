package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAuthProxyInjectsRendererHeader(t *testing.T) {
	var sawValue string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawValue = r.Header.Get("x-dsh-desktop-renderer")
		if r.Header.Get("X-DSH-Auth-Proxy") != "1" {
			t.Errorf("missing X-DSH-Auth-Proxy marker")
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()

	proxy := NewAuthProxy()
	local, err := proxy.StartListening(upstream.URL+"/ui?x=1", "x-dsh-desktop-renderer", "tok123")
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	resp, err := http.Get(local)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
	if sawValue != "tok123" {
		t.Fatalf("header not injected: value=%q", sawValue)
	}
	if !strings.Contains(PlatformAuthCapability(), "AuthProxy") {
		t.Fatal("expected platform notes")
	}
	if !strings.Contains(proxy.Status(), "production-required") {
		t.Fatalf("status=%s", proxy.Status())
	}
	if !strings.Contains(local, "/ui?x=1") {
		t.Fatalf("non-auth query should preserve path/query: %s", local)
	}
}

func TestAuthProxyRejectsNonLoopback(t *testing.T) {
	proxy := NewAuthProxy()
	_, err := proxy.StartListening("https://example.com/", "x-dsh-desktop-renderer", "tok")
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected loopback error, got %v", err)
	}
}

func TestAuthProxyBootstrapTokenAndInjectsCookie(t *testing.T) {
	var proxiedHadCookie atomic.Bool
	var bootstrapHits atomic.Int32
	var rootHits atomic.Int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-dsh-desktop-renderer") != "rend-tok" {
			http.Error(w, "missing renderer", http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/" && r.URL.Query().Get("token") == "exchange-secret":
			bootstrapHits.Add(1)
			http.SetCookie(w, &http.Cookie{
				Name:     "dsh_session",
				Value:    "session-abc",
				Path:     "/",
				HttpOnly: true,
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		case r.URL.Path == "/":
			rootHits.Add(1)
			c, err := r.Cookie("dsh_session")
			if err != nil || c.Value != "session-abc" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if r.Header.Get("X-DSH-Auth-Proxy") == "1" {
				proxiedHadCookie.Store(true)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<html><body>host-ui-ok</body></html>")
			return
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	proxy := NewAuthProxy()
	authURL := upstream.URL + "/?token=exchange-secret"
	local, err := proxy.StartListening(authURL, "x-dsh-desktop-renderer", "rend-tok")
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	if strings.Contains(local, "token=") {
		t.Fatalf("proxy URL must not expose token query: %s", local)
	}
	if !strings.HasSuffix(local, "/") || strings.Contains(local, "?") {
		t.Fatalf("expected bare root proxy URL, got %s", local)
	}
	if bootstrapHits.Load() < 1 {
		t.Fatal("expected bootstrap GET of token URL")
	}
	if !strings.Contains(proxy.Status(), "cookies=") {
		t.Fatalf("status should report cookie count: %s", proxy.Status())
	}

	resp, err := http.Get(local)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("proxied status=%d body=%q", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "host-ui-ok") {
		t.Fatalf("expected host HTML, got %q", body)
	}
	if !proxiedHadCookie.Load() {
		t.Fatal("expected AuthProxy to inject jar cookie on proxied request")
	}
	if rootHits.Load() < 1 {
		t.Fatal("expected at least one authenticated root hit via proxy")
	}
	// Set-Cookie from upstream should be stripped for the browser; jar is primary.
	if len(resp.Cookies()) != 0 {
		t.Fatalf("expected Set-Cookie stripped from proxy response, got %#v", resp.Cookies())
	}
}

func TestAuthProxyBootstrapFailsWithoutValidToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer upstream.Close()

	proxy := NewAuthProxy()
	_, err := proxy.StartListening(upstream.URL+"/?token=bad", "x-dsh-desktop-renderer", "rend-tok")
	if err == nil || !strings.Contains(err.Error(), "bootstrap") {
		t.Fatalf("expected bootstrap error, got %v", err)
	}
}

func TestAuthProxyRewritesOriginAndReferer(t *testing.T) {
	var sawOrigin, sawReferer string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawOrigin = r.Header.Get("Origin")
		sawReferer = r.Header.Get("Referer")
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()

	proxy := NewAuthProxy()
	local, err := proxy.StartListening(upstream.URL+"/", "x-dsh-desktop-renderer", "tok123")
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	req, err := http.NewRequest(http.MethodGet, local, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate WebKit loading the AuthProxy port (not the upstream Host port).
	req.Header.Set("Origin", "http://127.0.0.1:46609")
	req.Header.Set("Referer", "http://127.0.0.1:46609/settings")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	wantOrigin := strings.TrimRight(upstream.URL, "/")
	if sawOrigin != wantOrigin {
		t.Fatalf("Origin not rewritten: got %q want %q", sawOrigin, wantOrigin)
	}
	if !strings.HasPrefix(sawReferer, wantOrigin+"/") {
		t.Fatalf("Referer not rewritten: got %q want prefix %q", sawReferer, wantOrigin+"/")
	}
}
