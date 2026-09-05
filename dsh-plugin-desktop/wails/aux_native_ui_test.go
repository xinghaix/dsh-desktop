package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestRecoveryNativeStateOmitsNoticeNull(t *testing.T) {
	enc := recoveryNativeState("host failed", []string{"default"}, nil)
	if enc == "" {
		t.Fatal("empty state")
	}
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if strings.Contains(string(raw), `"notice":null`) {
		t.Fatalf("notice:null crashes RecoveryNoticeSurface deps; payload=%s", raw)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := payload["notice"]; ok {
		t.Fatalf("notice key must be omitted, got %#v", payload["notice"])
	}
	if payload["failureDetail"] != "host failed" {
		t.Fatalf("failureDetail=%v", payload["failureDetail"])
	}
	if payload["activeTab"] != "quick" {
		t.Fatalf("activeTab=%v", payload["activeTab"])
	}
	diags, _ := payload["diagnostics"].(map[string]any)
	if diags["status"] != "failed" {
		t.Fatalf("diagnostics=%v", payload["diagnostics"])
	}
}
