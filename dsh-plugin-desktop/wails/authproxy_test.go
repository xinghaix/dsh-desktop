package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthProxyInjectsRendererHeader(t *testing.T) {
	var sawName, sawValue string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawName = "x-dsh-desktop-renderer"
		sawValue = r.Header.Get("x-dsh-desktop-renderer")
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
	if sawName == "" || sawValue != "tok123" {
		t.Fatalf("header not injected: value=%q", sawValue)
	}
	if PlatformAuthCapability() == "" {
		t.Fatal("expected platform notes")
	}
}
