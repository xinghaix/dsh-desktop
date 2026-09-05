package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultHostBootstrapPrefersHostMain(t *testing.T) {
	dir := t.TempDir()
	plugin := filepath.Join(dir, "dsh-plugin-desktop")
	wailsDir := filepath.Join(plugin, "wails")
	if err := os.MkdirAll(wailsDir, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(wailsDir, "hostsidecar.go"), []byte("package main\n"), 0o644); err != nil { t.Fatal(err) }
	lib := filepath.Join(plugin, "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil { t.Fatal(err) }
	hostMain := filepath.Join(lib, "host-main.js")
	if err := os.WriteFile(hostMain, []byte("ok\n"), 0o644); err != nil { t.Fatal(err) }
	oldWD, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(wailsDir); err != nil { t.Fatal(err) }
	cmd, _, err := defaultHostBootstrap()
	if err != nil { t.Fatal(err) }
	if !strings.Contains(cmd, "host-main.js") { t.Fatalf("expected host-main.js, got %q", cmd) }
	if !strings.Contains(cmd, "DSH_WAILS_HOST_SIDECAR=1") { t.Fatalf("expected env, got %q", cmd) }
}
