package main

import (
	"os"
	"testing"
)

// P1 deeper smoke probes for Linux bed (notify / export / terminal / update).
func TestP1CapabilityProbes(t *testing.T) {
	caps := NewCapabilitiesService(nil, nil)

	res := caps.CheckForUpdates()
	t.Logf("CheckForUpdates: status=%q detail=%q canDownload=%v current=%q latest=%q",
		res.Status, res.Detail, res.CanDownload, res.CurrentHint, res.LatestHint)
	if res.Status == "" {
		t.Fatalf("expected non-empty update status")
	}
	if !res.CanDownload {
		t.Fatalf("linux AppImage download path should be advertised")
	}

	path, err := caps.ExportTextFile("dsh-p1-probe.txt", "p1 probe\n")
	t.Logf("ExportTextFile path=%q err=%v", path, err)
	// Headless beds often lack a dialog backend; graceful error is OK.

	err = caps.OpenTerminal(os.TempDir())
	t.Logf("OpenTerminal err=%v", err)
	// Missing terminal emulator is an expected graceful error on this bed.

	err = caps.NotifyAttention("dsh-p1-probe", "capabilities notify")
	t.Logf("NotifyAttention err=%v", err)
}
