package main

import (
	"strings"
	"testing"
)

func TestLanHttpsAnnounceIngestAndStatus(t *testing.T) {
	caps := NewCapabilitiesService(nil, nil)
	awaiting := caps.LanHttpsStatus()
	if !strings.Contains(awaiting, "awaiting-host-announce") {
		t.Fatalf("expected awaiting status, got %q", awaiting)
	}
	if caps.IngestLanHttpsAnnounceLine("noise") {
		t.Fatal("noise line should not ingest")
	}
	if caps.IngestLanHttpsAnnounceLine("DSH_HOST_LAN_HTTPS ") {
		t.Fatal("empty payload should not ingest")
	}
	line := "DSH_HOST_LAN_HTTPS state=ready port=8443 addresses=192.168.1.10 fingerprint=abcd error=null urls=https://192.168.1.10:8443/"
	if !caps.IngestLanHttpsAnnounceLine(line) {
		t.Fatal("expected announce ingest")
	}
	status := caps.LanHttpsStatus()
	if !strings.HasPrefix(status, "lan-https=announced ") {
		t.Fatalf("expected announced prefix, got %q", status)
	}
	if strings.Contains(status, "lan-https=lan-https=") {
		t.Fatalf("double prefix regression: %q", status)
	}
	if !strings.Contains(status, "state=ready") || !strings.Contains(status, "port=8443") {
		t.Fatalf("missing announce fields: %q", status)
	}
	agg := caps.CapabilitiesStatus()
	if !strings.Contains(agg, "lan-https=announced ") {
		t.Fatalf("CapabilitiesStatus missing lan announce: %q", agg)
	}
	if !strings.Contains(agg, "packaging=electron-builder-default-product-ci") {
		t.Fatalf("CapabilitiesStatus missing packaging note: %q", agg)
	}
}
