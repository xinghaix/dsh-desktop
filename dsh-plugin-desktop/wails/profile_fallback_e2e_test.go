package main

import (
	"path/filepath"
	"testing"
)

func TestE2EProfileFallbackWeb(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg"))
	t.Setenv("DSH_HOME", filepath.Join(home, ".dsh"))
	t.Setenv("DSH_PROFILE", "")
	t.Setenv("DSH_DESKTOP_DEFAULT_PROFILE", "")
	if got := resolveBootstrapProfile(); got != "web" {
		t.Fatalf("want web, got %q", got)
	}
}

func TestClassifyProfileCreateFailure(t *testing.T) {
	got := classifyHostFailure(errString("failed to create web profile"), nil)
	if got != hostFailProfile {
		t.Fatalf("want profile, got %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
