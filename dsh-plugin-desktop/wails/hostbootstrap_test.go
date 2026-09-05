package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)


func isolateNoUserHost(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DSH_BIN", "")
	t.Setenv("DSH_HOME", "")
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("PATH", "/usr/bin:/bin")
	return home
}

func TestDefaultHostBootstrapPrefersHostMain(t *testing.T) {
	_ = isolateNoUserHost(t)
	dir := t.TempDir()
	plugin := filepath.Join(dir, "dsh-plugin-desktop")
	wailsDir := filepath.Join(plugin, "wails")
	if err := os.MkdirAll(wailsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wailsDir, "hostsidecar.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(plugin, "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	hostMain := filepath.Join(lib, "host-main.js")
	if err := os.WriteFile(hostMain, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(wailsDir); err != nil {
		t.Fatal(err)
	}
	cmd, _, err := defaultHostBootstrap()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "host-main.js") {
		t.Fatalf("expected host-main.js, got %q", cmd)
	}
	if !strings.Contains(cmd, "DSH_WAILS_HOST_SIDECAR=1") {
		t.Fatalf("expected env, got %q", cmd)
	}
}
func TestResolveHostLauncherModeGoEnv(t *testing.T) {
	t.Setenv("DSH_HOST_LAUNCHER", "electron-as-node")
	mode, reason := resolveHostLauncherModeGo(true, "/bin/electron")
	if mode != hostLauncherElectronAsNode {
		t.Fatalf("mode=%v reason=%s", mode, reason)
	}
	t.Setenv("DSH_HOST_LAUNCHER", "node")
	mode, _ = resolveHostLauncherModeGo(true, "/bin/electron")
	if mode != hostLauncherNode {
		t.Fatalf("expected node, got %v", mode)
	}
	t.Setenv("DSH_HOST_LAUNCHER", "")
	t.Setenv("DSH_HOST_ELECTRON_AS_NODE", "1")
	mode, _ = resolveHostLauncherModeGo(true, "/bin/electron")
	if mode != hostLauncherElectronAsNode {
		t.Fatalf("expected electron-as-node from flag, got %v", mode)
	}
}

func TestDefaultHostBootstrapElectronAsNode(t *testing.T) {
	_ = isolateNoUserHost(t)
	dir := t.TempDir()
	plugin := filepath.Join(dir, "dsh-plugin-desktop")
	wailsDir := filepath.Join(plugin, "wails")
	if err := os.MkdirAll(wailsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wailsDir, "hostsidecar.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(plugin, "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	hostMain := filepath.Join(lib, "host-main.js")
	if err := os.WriteFile(hostMain, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	electron := filepath.Join(dir, "fake-electron")
	if err := os.WriteFile(electron, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_HOST_LAUNCHER", "electron-as-node")
	t.Setenv("ELECTRON_PATH", electron)
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(wailsDir); err != nil {
		t.Fatal(err)
	}
	cmd, _, err := defaultHostBootstrap()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "fake-electron") {
		t.Fatalf("expected electron path, got %q", cmd)
	}
	if !strings.Contains(cmd, "ELECTRON"+"_RUN_AS_NODE=1") {
		t.Fatalf("expected run-as-node, got %q", cmd)
	}
	if !strings.Contains(cmd, "host-main.js") {
		t.Fatalf("expected host-main, got %q", cmd)
	}
}

func TestCommandFromHostBinJS(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "host-main.js")
	if err := os.WriteFile(js, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, err := commandFromHostBin(js)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "node") || !strings.Contains(cmd, "host-main.js") || !strings.Contains(cmd, hostSidecarArgument) {
		t.Fatalf("unexpected cmd %q", cmd)
	}
}

func TestCommandFromHostBinExecutable(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "dsh-desktop")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd, err := commandFromHostBin(bin)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "dsh-desktop") || !strings.Contains(cmd, hostSidecarArgument) {
		t.Fatalf("unexpected cmd %q", cmd)
	}
}

func TestDefaultHostBootstrapHonorsDSHBin(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "custom-host.js")
	if err := os.WriteFile(js, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_BIN", js)
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	cmd, _, err := defaultHostBootstrap()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "custom-host.js") {
		t.Fatalf("expected DSH_BIN entry, got %q", cmd)
	}
}

func TestDefaultHostBootstrapPathFallback(t *testing.T) {
	dir := t.TempDir()
	shim := filepath.Join(dir, "dsh-desktop")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_BIN", "")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin:/bin")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	cmd, _, err := defaultHostBootstrap()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "dsh-desktop") {
		t.Fatalf("expected PATH dsh-desktop, got %q", cmd)
	}
}


func TestProbeHostDiscoveryFromDSHHome(t *testing.T) {
	home := isolateNoUserHost(t)
	dshHome := filepath.Join(home, "my-dsh-home")
	binDir := filepath.Join(dshHome, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(binDir, "dsh-desktop")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_HOME", dshHome)
	rep := ProbeHostDiscovery()
	if rep.Hit == nil {
		t.Fatalf("expected hit, message=%s checked=%v", rep.Message, rep.Checked)
	}
	if !strings.Contains(rep.Hit.Reason, "DSH_HOME") {
		t.Fatalf("reason=%q", rep.Hit.Reason)
	}
	if !strings.Contains(rep.Hit.Path, "dsh-desktop") {
		t.Fatalf("path=%q", rep.Hit.Path)
	}
	// Without packaged layout, bootstrap uses user-home Host as escape hatch.
	empty := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(empty); err != nil { t.Fatal(err) }
	cmd, _, err := defaultHostBootstrap()
	if err != nil { t.Fatal(err) }
	if !strings.Contains(cmd, "dsh-desktop") { t.Fatalf("cmd=%q", cmd) }
}

func TestProbeHostDiscoveryFromDotDshHome(t *testing.T) {
	home := isolateNoUserHost(t)
	dot := filepath.Join(home, ".dsh")
	lib := filepath.Join(dot, "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	hostMain := filepath.Join(lib, "host-main.js")
	if err := os.WriteFile(hostMain, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := ProbeHostDiscovery()
	if rep.Hit == nil {
		t.Fatalf("expected ~/.dsh hit, msg=%s checked=%v", rep.Message, rep.Checked)
	}
	if !strings.Contains(rep.Hit.Path, hostMain) && !strings.Contains(rep.Hit.Path, "host-main.js") {
		t.Fatalf("path=%q", rep.Hit.Path)
	}
	if !strings.Contains(rep.Hit.Reason, ".dsh") {
		t.Fatalf("reason=%q", rep.Hit.Reason)
	}
}

func TestProbeHostDiscoveryMissingFriendlyMessage(t *testing.T) {
	_ = isolateNoUserHost(t)
	// Also leave monorepo: chdir to empty temp so layout missing
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	rep := ProbeHostDiscovery()
	if rep.Hit != nil {
		t.Fatalf("unexpected hit %+v", rep.Hit)
	}
	if !strings.Contains(rep.Message, "Checked paths") && !strings.Contains(rep.Message, "已检查") {
		t.Fatalf("expected checked paths in message: %s", rep.Message)
	}
	if !strings.Contains(rep.Message, "DSH_HOME") || !strings.Contains(rep.Message, "DSH_BIN") {
		t.Fatalf("expected env hints: %s", rep.Message)
	}
	_, _, err = defaultHostBootstrap()
	if err == nil {
		t.Fatal("expected bootstrap error")
	}
	if !strings.Contains(err.Error(), "DSH_BIN") {
		t.Fatalf("bootstrap err=%v", err)
	}
}

func TestPackagedBeatsUserHome(t *testing.T) {
	home := isolateNoUserHost(t)
	// Fake monorepo layout in cwd
	dir := t.TempDir()
	plugin := filepath.Join(dir, "dsh-plugin-desktop")
	wailsDir := filepath.Join(plugin, "wails")
	if err := os.MkdirAll(wailsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wailsDir, "hostsidecar.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(plugin, "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "host-main.js"), []byte("monorepo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// User home install
	userLib := filepath.Join(home, "dsh", "lib")
	if err := os.MkdirAll(userLib, 0o755); err != nil {
		t.Fatal(err)
	}
	userHost := filepath.Join(userLib, "host-main.js")
	if err := os.WriteFile(userHost, []byte("user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(wailsDir); err != nil {
		t.Fatal(err)
	}
	cmd, _, err := defaultHostBootstrap()
	if err != nil {
		t.Fatal(err)
	}
	monoHost := filepath.Join(plugin, "lib", "host-main.js")
	if !strings.Contains(cmd, monoHost) {
		t.Fatalf("expected packaged/monorepo host, got %q", cmd)
	}
	if strings.Contains(cmd, userHost) {
		t.Fatalf("should not prefer user home when packaged Host exists: %q", cmd)
	}
}

func TestDSHBinBeatsDSHHome(t *testing.T) {
	home := isolateNoUserHost(t)
	dshHome := filepath.Join(home, ".dsh")
	if err := os.MkdirAll(filepath.Join(dshHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dshHome, "bin", "dsh-desktop"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_HOME", dshHome)
	js := filepath.Join(home, "explicit-host.js")
	if err := os.WriteFile(js, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_BIN", js)
	rep := ProbeHostDiscovery()
	if rep.Hit == nil || rep.Hit.Reason != "DSH_BIN" {
		t.Fatalf("expected DSH_BIN, got %+v", rep.Hit)
	}
}
