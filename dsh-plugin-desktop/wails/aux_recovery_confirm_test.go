package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckpointConfirmCopyDoesNotExecute(t *testing.T) {
	title, message, confirm := checkpointConfirmCopy(&checkpointPreview{
		PreviewID:  "preview_abc",
		SlotID:     "slot-1",
		CapturedAt: "2026-09-05T01:00:00Z",
		ExpiresAt:  "2026-09-05T01:05:00Z",
	})
	if title == "" || confirm == "" {
		t.Fatalf("empty copy: %q %q", title, confirm)
	}
	if !strings.Contains(message, "slot-1") || !strings.Contains(message, "2026-09-05T01:00:00Z") {
		t.Fatalf("message missing slot/time: %q", message)
	}
	if strings.Contains(strings.ToLower(message), "execut") && strings.Contains(message, "silently") {
		t.Fatalf("unexpected silent execute wording: %q", message)
	}
}

func TestUninstallConfirmCopy(t *testing.T) {
	title, message, confirm := uninstallConfirmCopy(&uninstallPreview{
		PreviewID:   "u1",
		PackageName: "dsh-plugin-example",
		ExpiresAt:   "soon",
	})
	if title == "" || confirm != "Uninstall" {
		t.Fatalf("bad copy: %q %q", title, confirm)
	}
	if !strings.Contains(message, "dsh-plugin-example") {
		t.Fatalf("message missing package: %q", message)
	}
}

func TestRecoveryConfigPathSettings(t *testing.T) {
	ud := resolveDesktopUserDataDir()
	if ud == "" {
		t.Fatal("expected desktop user data dir")
	}
	got := recoveryConfigPath("open-settings-document", "")
	if !strings.HasSuffix(got, "settings.yaml") {
		t.Fatalf("settings path: %q", got)
	}
	dir := recoveryConfigPath("open-profile-directory", "default")
	if dir == "" {
		t.Fatal("expected profile directory path")
	}
}

func TestPendingConfirmTakeClears(t *testing.T) {
	aux := NewAuxWindowService(nil)
	aux.storePendingConfirm(&pendingRecoveryConfirm{Kind: "checkpoint", PreviewID: "p1"})
	got := aux.takePendingConfirm()
	if got == nil || got.PreviewID != "p1" {
		t.Fatalf("take failed: %+v", got)
	}
	if aux.takePendingConfirm() != nil {
		t.Fatal("expected clear after take")
	}
}

func TestHandleCheckpointPreviewRequiresConfirmBeforeExecute(t *testing.T) {
	executeCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/checkpoint/preview", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"previewId":  "preview_test_1",
			"slotId":     "slot-1",
			"capturedAt": "2026-09-05T01:00:00Z",
			"expiresAt":  "2026-09-05T01:05:00Z",
		})
	})
	mux.HandleFunc("/v1/checkpoint/execute", func(w http.ResponseWriter, r *http.Request) {
		executeCalled = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	aux := NewAuxWindowService(nil)
	aux.AttachRecoveryRPC(&RecoveryRpcClient{
		BaseURL:    server.URL + "/",
		Token:      "tok",
		HTTPClient: server.Client(),
	})
	err := aux.handleCheckpointPreview("slot-1")
	// Without an attached Wails app, OpenConfirmDialog fails — that is OK.
	// The critical assertion is that execute was never called.
	_ = err
	if executeCalled {
		t.Fatal("checkpoint execute must not run before user confirm")
	}
	pending := aux.takePendingConfirm()
	if pending == nil || pending.Kind != "checkpoint" || pending.PreviewID != "preview_test_1" {
		t.Fatalf("expected pending checkpoint confirm, got %+v", pending)
	}
}

func TestHandleUninstallPreviewRequiresConfirmBeforeExecute(t *testing.T) {
	executeCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/uninstall/preview", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"previewId":   "u_preview_1",
			"packageName": "dsh-plugin-example",
			"expiresAt":   "soon",
		})
	})
	mux.HandleFunc("/v1/uninstall/execute", func(w http.ResponseWriter, r *http.Request) {
		executeCalled = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	aux := NewAuxWindowService(nil)
	aux.AttachRecoveryRPC(&RecoveryRpcClient{
		BaseURL:    server.URL + "/",
		Token:      "tok",
		HTTPClient: server.Client(),
	})
	_ = aux.handleUninstallPreview("bundle_1")
	if executeCalled {
		t.Fatal("uninstall execute must not run before user confirm")
	}
	pending := aux.takePendingConfirm()
	if pending == nil || pending.Kind != "uninstall" || pending.PreviewID != "u_preview_1" {
		t.Fatalf("expected pending uninstall confirm, got %+v", pending)
	}
}

func TestCompleteConfirmDialogCancelDoesNotExecute(t *testing.T) {
	executeCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/checkpoint/execute", func(w http.ResponseWriter, r *http.Request) {
		executeCalled = true
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	aux := NewAuxWindowService(nil)
	aux.AttachRecoveryRPC(&RecoveryRpcClient{BaseURL: server.URL + "/", Token: "tok", HTTPClient: server.Client()})
	aux.storePendingConfirm(&pendingRecoveryConfirm{Kind: "checkpoint", PreviewID: "p1"})
	if err := aux.CompleteConfirmDialog("cancel"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if executeCalled {
		t.Fatal("cancel must not execute")
	}
	if aux.takePendingConfirm() != nil {
		t.Fatal("pending should be cleared on cancel")
	}
}
