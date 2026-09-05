package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyHostFailureKinds(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fmt.Errorf("timed out waiting for Cordis Host UI URL"), hostFailTimeout},
		{fmt.Errorf("listen tcp 127.0.0.1:43124: bind: address already in use"), hostFailPortBind},
		{fmt.Errorf("EADDRINUSE: address already in use"), hostFailPortBind},
		{fmt.Errorf("Host exited unexpectedly: signal: killed"), hostFailCrash},
		{fmt.Errorf("failed to start host: executable file not found"), hostFailStart},
		{fmt.Errorf("profile web create failed"), hostFailProfile},
	}
	for _, tc := range cases {
		got := classifyHostFailure(tc.err, nil)
		if got != tc.want {
			t.Fatalf("classify(%v)=%q want %q", tc.err, got, tc.want)
		}
	}
}

func TestClassifyMissingVsInvalidHome(t *testing.T) {
	t.Setenv("DSH_BIN", "")
	t.Setenv("DSH_HOME", "")
	t.Setenv("HOME", t.TempDir())
	rep := ProbeHostDiscovery()
	err := fmt.Errorf("%s", rep.Message)
	if got := classifyHostFailure(err, &rep); got != hostFailMissing {
		t.Fatalf("expected missing, got %s", got)
	}

	bad := filepath.Join(t.TempDir(), "empty-home")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSH_HOME", bad)
	rep2 := ProbeHostDiscovery()
	err2 := fmt.Errorf("%s", rep2.Message)
	got := classifyHostFailure(err2, &rep2)
	if got != hostFailNotUsable && got != hostFailInvalidHome {
		t.Fatalf("expected not-usable/invalid-home, got %s msg=%s", got, rep2.Message)
	}
	if !strings.Contains(rep2.Message, "DSH_HOME") {
		t.Fatalf("expected DSH_HOME hint in message")
	}
}

func TestValidateChosenDshHome(t *testing.T) {
	empty := t.TempDir()
	if err := validateChosenDshHome(empty); err == nil {
		t.Fatal("expected error for empty install root")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(bin, "dsh-desktop")
	if err := os.WriteFile(entry, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateChosenDshHome(root); err != nil {
		t.Fatalf("expected usable root, got %v", err)
	}
}

func TestHostErrorPageURL(t *testing.T) {
	u := hostErrorPageURL(hostFailTimeout, "timed out waiting")
	if !strings.HasPrefix(u, "/shell-ui/host-error.html?") {
		t.Fatalf("bad url %s", u)
	}
	if !strings.Contains(u, "kind=timeout") {
		t.Fatalf("missing kind: %s", u)
	}
}
