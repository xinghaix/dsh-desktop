package main

import (
	"strings"
	"testing"
)

func TestNormalizeRecoveryActionMapsDebt(t *testing.T) {
	if got := normalizeRecoveryAction("enter-safe-mode"); got != "safe-mode" {
		t.Fatalf("enter-safe-mode -> %q", got)
	}
	// Checkpoint/uninstall keep their action names; CompleteRecovery routes to RPC or debt.
	if got := normalizeRecoveryAction("preview-checkpoint"); got != "preview-checkpoint" {
		t.Fatalf("preview-checkpoint -> %q", got)
	}
	if got := normalizeRecoveryAction("preview-uninstall"); got != "preview-uninstall" {
		t.Fatalf("preview-uninstall -> %q", got)
	}
	msg := recoveryDebtMessage("debt-checkpoint")
	if !strings.Contains(msg, "snapshot()") || !strings.Contains(msg, "DSH_HOST_RECOVERY_RPC") || !strings.Contains(msg, "executeCheckpointRestore") {
		t.Fatalf("checkpoint debt message incomplete: %q", msg)
	}
	msg = recoveryDebtMessage("debt-uninstall")
	if !strings.Contains(msg, "previewUninstall") || !strings.Contains(msg, "executeUninstall") {
		t.Fatalf("uninstall debt message incomplete: %q", msg)
	}
}

func TestParseRecoveryRpcAnnounceLine(t *testing.T) {
	u, tok, ok := ParseRecoveryRpcAnnounceLine("DSH_HOST_RECOVERY_RPC http://127.0.0.1:9/ token=abc")
	if !ok || u != "http://127.0.0.1:9/" || tok != "abc" {
		t.Fatalf("parse failed: %v %q %q", ok, u, tok)
	}
	if _, _, ok := ParseRecoveryRpcAnnounceLine("DSH_HOST_READY http://127.0.0.1:1/"); ok {
		t.Fatal("ready line must not parse as recovery rpc")
	}
}
