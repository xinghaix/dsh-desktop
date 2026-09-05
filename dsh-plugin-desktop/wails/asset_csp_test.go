package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestNativeUICSPAllowsInline(t *testing.T) {
	h := application.AssetFileServerFS(assets)
	req := httptest.NewRequest(http.MethodGet, "/shell-ui/native-ui/recovery.html", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Body)
	s := string(body)
	if strings.Contains(s, "connect-src 'none'") {
		t.Fatal("still has connect-src none")
	}
	if !strings.Contains(s, "script-src 'self' 'unsafe-inline'") {
		t.Fatalf("missing relaxed CSP: %s", s[:min(400, len(s))])
	}
}
