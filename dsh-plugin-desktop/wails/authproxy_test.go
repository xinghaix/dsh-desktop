package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
}

func TestAuthProxyRejectsNonLoopback(t *testing.T) {
	proxy := NewAuthProxy()
	_, err := proxy.StartListening("https://example.com/", "x-dsh-desktop-renderer", "tok")
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected loopback error, got %v", err)
	}
}
