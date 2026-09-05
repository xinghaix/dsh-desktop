package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecoveryRpcClientExportDiagnosticsAndQuiesce(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/diagnostics/export", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": "/tmp/diagnostics-unit.zip"})
	})
	mux.HandleFunc("/v1/quiesce", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "detail": "quiesce=ok (mock)"})
	})
	mux.HandleFunc("/v1/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "action": "restart",
			"quiesce": map[string]any{"ok": true, "detail": "quiesce=ok (mock)"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &RecoveryRpcClient{
		BaseURL:    server.URL + "/",
		Token:      "tok",
		HTTPClient: server.Client(),
	}
	ctx := t.Context()
	exp, err := client.ExportDiagnostics(ctx)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if exp.Path != "/tmp/diagnostics-unit.zip" {
		t.Fatalf("path: %q", exp.Path)
	}
	q, err := client.Quiesce(ctx)
	if err != nil || !q.OK || !strings.Contains(q.Detail, "quiesce=ok") {
		t.Fatalf("quiesce: %+v %v", q, err)
	}
	if err := client.Complete(ctx, "restart"); err != nil {
		t.Fatalf("complete: %v", err)
	}
}

func TestQuiesceHostBestEffortWithoutRPC(t *testing.T) {
	aux := NewAuxWindowService(nil)
	note := aux.quiesceHostBestEffort(50 * time.Millisecond)
	if !strings.Contains(note, "no Recovery RPC") {
		t.Fatalf("expected skip note, got %q", note)
	}
}

func TestQuiesceHostBestEffortWithRPC(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/quiesce", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "detail": "quiesce=ok (test)"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	aux := NewAuxWindowService(nil)
	aux.AttachRecoveryRPC(&RecoveryRpcClient{
		BaseURL:    server.URL + "/",
		Token:      "tok",
		HTTPClient: server.Client(),
	})
	note := aux.quiesceHostBestEffort(2 * time.Second)
	if note != "quiesce=ok (test)" {
		t.Fatalf("note=%q", note)
	}
}

func TestCopyFileBestEffort(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.zip")
	dst := filepath.Join(dir, "out", "b.zip")
	if err := os.WriteFile(src, []byte("diag-zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileBestEffort(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "diag-zip" {
		t.Fatalf("got %q", got)
	}
}

func TestDebtDiagnosticsMessageMentionsExport(t *testing.T) {
	msg := recoveryDebtMessage("debt-diagnostics")
	if !strings.Contains(msg, "/v1/diagnostics/export") || !strings.Contains(msg, "--export-diagnostics") {
		t.Fatalf("debt message incomplete: %q", msg)
	}
}
