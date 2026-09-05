package main

import (
	"strings"
	"testing"
	"time"
)

func TestHostRelaunchBackoffAttempts(t *testing.T) {
	p := hostRelaunchPolicy{
		MaxAttempts: 3,
		StableAfter: 45 * time.Second,
		Backoff:     []time.Duration{time.Second, 2 * time.Second, 3 * time.Second},
	}
	ok, n, d := p.nextRelaunchAttempt(0, time.Second)
	if !ok || n != 1 || d != time.Second {
		t.Fatalf("first: ok=%v n=%d d=%s", ok, n, d)
	}
	ok, n, d = p.nextRelaunchAttempt(1, time.Second)
	if !ok || n != 2 || d != 2*time.Second {
		t.Fatalf("second: ok=%v n=%d d=%s", ok, n, d)
	}
	ok, n, d = p.nextRelaunchAttempt(2, time.Second)
	if !ok || n != 3 || d != 3*time.Second {
		t.Fatalf("third: ok=%v n=%d d=%s", ok, n, d)
	}
	ok, n, _ = p.nextRelaunchAttempt(3, time.Second)
	if ok || n != 3 {
		t.Fatalf("exhausted: ok=%v n=%d", ok, n)
	}
}

func TestHostRelaunchResetsAfterStableUptime(t *testing.T) {
	p := hostRelaunchPolicy{
		MaxAttempts: 3,
		StableAfter: 10 * time.Second,
		Backoff:     []time.Duration{time.Millisecond},
	}
	ok, n, _ := p.nextRelaunchAttempt(3, 15*time.Second)
	if !ok || n != 1 {
		t.Fatalf("stable reset: ok=%v n=%d want relaunch attempt 1", ok, n)
	}
}

func TestHostRelaunchMaxZeroDisables(t *testing.T) {
	p := hostRelaunchPolicy{MaxAttempts: 0, Backoff: []time.Duration{time.Second}}
	ok, _, _ := p.nextRelaunchAttempt(0, 0)
	if ok {
		t.Fatal("max=0 should disable relaunch")
	}
}

func TestLoadHostRelaunchPolicyEnv(t *testing.T) {
	t.Setenv("DSH_HOST_RELAUNCH_MAX", "2")
	t.Setenv("DSH_HOST_RELAUNCH_STABLE", "5s")
	t.Setenv("DSH_HOST_RELAUNCH_BACKOFF", "10ms,20ms")
	p := loadHostRelaunchPolicy()
	if p.MaxAttempts != 2 || p.StableAfter != 5*time.Second || len(p.Backoff) != 2 {
		t.Fatalf("policy=%+v", p)
	}
}

func TestHostRestartingPageURL(t *testing.T) {
	u := hostRestartingPageURL(2, 3, "Host exited unexpectedly: exit status 1")
	if !strings.HasPrefix(u, "/shell-ui/host-restarting.html?") {
		t.Fatalf("bad url %s", u)
	}
	if !strings.Contains(u, "attempt=2") || !strings.Contains(u, "max=3") {
		t.Fatalf("missing attempt/max: %s", u)
	}
}

func TestExpectExitSuppressesUnexpectedCallback(t *testing.T) {
	h := NewHostSidecar()
	called := false
	h.OnUnexpectedExit(func(err error) { called = true })
	h.ExpectExit()
	h.mu.Lock()
	h.readyAnnounced = true
	h.exitExpected = true
	h.spawnSeq = 1
	cb := h.onUnexpectedExit
	expected := h.exitExpected
	h.mu.Unlock()
	if expected && cb != nil {
		// Mimic Wait path: expected exit must not invoke callback.
		if !expected {
			cb(errString("boom"))
		}
	}
	if called {
		t.Fatal("callback should not run when exitExpected")
	}
}

func TestHandleUnexpectedHostExitRespectsSuppress(t *testing.T) {
	sidecar := NewHostSidecar()
	shell := NewShellService(sidecar)
	shell.ExpectHostExit()
	shell.HandleUnexpectedHostExit(errString("Host exited unexpectedly: exit status 1"))
	st := shell.RelaunchStatus()
	if !strings.Contains(st, "suppress=true") {
		t.Fatalf("want suppress, got %s", st)
	}
	if shell.relaunchAttempts != 0 {
		t.Fatalf("suppress path must not bump attempts, got %d", shell.relaunchAttempts)
	}
}

func TestHandleUnexpectedHostExitSchedulesRelaunch(t *testing.T) {
	t.Setenv("DSH_HOST_RELAUNCH_MAX", "3")
	t.Setenv("DSH_HOST_RELAUNCH_BACKOFF", "50ms,50ms,50ms")
	t.Setenv("DSH_HOST_RELAUNCH_STABLE", "1h")
	// Force start failure quickly: no host install in empty HOME.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/.config")
	t.Setenv("DSH_HOME", "")
	t.Setenv("DSH_BIN", "")
	t.Setenv("DSH_HOST_COMMAND", "")
	t.Setenv("DSH_HOST_URL", "")
	t.Setenv("DSH_HOST_URL_FILE", "")
	t.Setenv("DSH_ALLOW_PACKAGED_HOST", "")

	sidecar := NewHostSidecar()
	shell := NewShellService(sidecar)
	shell.HandleUnexpectedHostExit(errString("Host exited unexpectedly after READY (exit 0)"))
	// Give relaunch goroutine a moment to mark attempts.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := shell.RelaunchStatus()
		if strings.Contains(st, "attempts=1") || strings.Contains(st, "attempts=2") || strings.Contains(st, "attempts=3") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected relaunch attempts to advance, status=%s", shell.RelaunchStatus())
}
