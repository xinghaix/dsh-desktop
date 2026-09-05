package main

import (
	"strings"
	"testing"
)

func TestNormalizeRecoveryActionMapsDebt(t *testing.T) {
	if got := normalizeRecoveryAction("enter-safe-mode"); got != "safe-mode" {
		t.Fatalf("enter-safe-mode -> %q", got)
	}
	if got := normalizeRecoveryAction("preview-checkpoint"); got != "debt-checkpoint" {
		t.Fatalf("preview-checkpoint -> %q", got)
	}
	if got := normalizeRecoveryAction("preview-uninstall"); got != "debt-uninstall" {
		t.Fatalf("preview-uninstall -> %q", got)
	}
	msg := recoveryDebtMessage("debt-checkpoint")
	if !strings.Contains(msg, "snapshot()") || !strings.Contains(msg, "Host↔Wails") || !strings.Contains(msg, "executeCheckpointRestore") {
		t.Fatalf("checkpoint debt message incomplete: %q", msg)
	}
	msg = recoveryDebtMessage("debt-uninstall")
	if !strings.Contains(msg, "previewUninstall") || !strings.Contains(msg, "executeUninstall") {
		t.Fatalf("uninstall debt message incomplete: %q", msg)
	}
}
