package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrashEvidenceBeginAndClean(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DSH_DESKTOP_USER_DATA", root)
	svc := NewCrashEvidenceService()
	note := svc.BeginRun("test-version")
	if !strings.Contains(note, "crash-evidence=active") {
		t.Fatalf("unexpected begin note: %s", note)
	}
	active := filepath.Join(root, "crash-evidence", "active-run.json")
	if _, err := os.Stat(active); err != nil {
		t.Fatal(err)
	}
	svc.MarkClean()
	if _, err := os.Stat(active); !os.IsNotExist(err) {
		t.Fatalf("expected active-run removed, err=%v", err)
	}
	status := svc.Status()
	if !strings.Contains(status, "clean") {
		t.Fatalf("status=%s", status)
	}
}

func TestCrashEvidencePanicDump(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DSH_DESKTOP_USER_DATA", root)
	svc := NewCrashEvidenceService()
	_ = svc.BeginRun("v")
	path := svc.WritePanicDump("boom")
	if path == "" {
		t.Fatal("expected panic dump path")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "boom") {
		t.Fatalf("dump missing recovered value: %s", raw)
	}
}

func TestCrashEvidenceDetectsPreviousMarker(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DSH_DESKTOP_USER_DATA", root)
	dir := filepath.Join(root, "crash-evidence")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "active-run.json"), []byte(`{"pid":1,"version":"old"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewCrashEvidenceService()
	note := svc.BeginRun("new")
	if !strings.Contains(note, "unclean") {
		t.Fatalf("expected unclean note, got %s", note)
	}
}
