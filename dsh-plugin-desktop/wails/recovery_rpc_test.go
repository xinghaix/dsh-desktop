package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecoveryRpcClientSnapshotRoundTrip(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "detail": "test", "hasController": false})
	})
	mux.HandleFunc("/v1/snapshot", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(RecoverySnapshot{
			ProfileName: "",
			Bundles:     []RecoveryBundle{},
			Checkpoints: []RecoveryCheckpoint{},
			Controller:  false,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &RecoveryRpcClient{BaseURL: server.URL + "/", Token: "test-token", HTTPClient: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if !health.OK || health.Detail != "test" {
		t.Fatalf("unexpected health: %+v", health)
	}
	snap, err := client.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Controller {
		t.Fatal("expected structural empty snapshot without controller")
	}
	if snap.Checkpoints == nil {
		t.Fatal("checkpoints must be non-nil slice")
	}
}

func TestRecoveryRpcClientUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Recovery RPC requires bearer token"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &RecoveryRpcClient{BaseURL: server.URL + "/", Token: "wrong", HTTPClient: server.Client()}
	_, err := client.Health(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
