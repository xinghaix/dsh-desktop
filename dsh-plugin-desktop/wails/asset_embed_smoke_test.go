package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestEmbeddedShellUIAssetsServed(t *testing.T) {
	var files []string
	_ = fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	hasShell, hasAuxDir := false, false
	for _, f := range files {
		if strings.Contains(f, "/shell-ui/") {
			hasShell = true
		}
		if strings.Contains(f, "/aux/") || strings.HasSuffix(f, "/aux") {
			hasAuxDir = true
		}
	}
	if !hasShell {
		t.Fatalf("shell-ui missing from go:embed (count=%d)", len(files))
	}
	if hasAuxDir {
		t.Fatal("legacy aux/ still embedded — go:embed rejects name aux on Windows reserved list")
	}

	h := application.AssetFileServerFS(assets)
	paths := []string{
		"/shell-ui/native-ui/recovery.html",
		"/shell-ui/native-ui/setup-wizard.html",
		"/shell-ui/native-ui/profile-selector.html",
		"/shell-ui/native-ui/assets/recovery-B5X2LwZo.js",
		"/shell-ui/native-ui/assets/setup-wizard-ChHWqYtN.js",
		"/shell-ui/native-ui/assets/profile-selector-b6o7KzZU.js",
		"/shell-ui/native-ui/assets/DesktopFrame-Dv-n-ZuF.js",
		"/shell-ui/native-ui/assets/DesktopFrame-B7VTinjh.css",
		"/shell-ui/recovery.html",
		"/shell-ui/status.html",
		"/shell-ui/host-install.html",
		"/shell-ui/host-error.html",
		"/shell-ui/host-restarting.html",
		"/index.html",
	}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("%s -> %d", p, rr.Code)
		}
	}
}

func TestResolveAuxURLUsesShellUI(t *testing.T) {
	a := NewAuxWindowService(nil)
	u := a.ResolveAuxURLForTest("recovery", url.Values{"locale": []string{"en"}})
	if !strings.HasPrefix(u, "/shell-ui/") {
		t.Fatalf("expected /shell-ui/ prefix, got %q", u)
	}
	if strings.Contains(u, "/aux/") {
		t.Fatalf("must not use /aux/ (unembeddable): %q", u)
	}
}
