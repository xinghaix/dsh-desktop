package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Host auto-relaunch policy after unexpected sidecar exit.
// Override via env for tests / ops:
//
//	DSH_HOST_RELAUNCH_MAX      — max attempts (default 3)
//	DSH_HOST_RELAUNCH_STABLE   — uptime that resets the counter (default 45s)
//	DSH_HOST_RELAUNCH_BACKOFF  — comma-separated durations (default 1s,2s,3s)
const (
	defaultHostRelaunchMax    = 3
	defaultHostRelaunchStable = 45 * time.Second
)

var defaultHostRelaunchBackoff = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	3 * time.Second,
}

type hostRelaunchPolicy struct {
	MaxAttempts int
	StableAfter time.Duration
	Backoff     []time.Duration
}

func loadHostRelaunchPolicy() hostRelaunchPolicy {
	p := hostRelaunchPolicy{
		MaxAttempts: defaultHostRelaunchMax,
		StableAfter: defaultHostRelaunchStable,
		Backoff:     append([]time.Duration(nil), defaultHostRelaunchBackoff...),
	}
	if v := strings.TrimSpace(os.Getenv("DSH_HOST_RELAUNCH_MAX")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.MaxAttempts = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("DSH_HOST_RELAUNCH_STABLE")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 0 {
			p.StableAfter = d
		}
	}
	if v := strings.TrimSpace(os.Getenv("DSH_HOST_RELAUNCH_BACKOFF")); v != "" {
		parts := strings.Split(v, ",")
		var out []time.Duration
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			d, err := time.ParseDuration(part)
			if err != nil || d < 0 {
				continue
			}
			out = append(out, d)
		}
		if len(out) > 0 {
			p.Backoff = out
		}
	}
	if len(p.Backoff) == 0 {
		p.Backoff = append([]time.Duration(nil), defaultHostRelaunchBackoff...)
	}
	return p
}

// nextRelaunchAttempt decides whether to auto-relaunch after an unexpected exit.
// attemptsSoFar is the number of relaunches already tried in this crash storm (0 = first crash).
// uptime is how long Host stayed READY before this exit.
//
// Returns (shouldRelaunch, attemptNumber 1-based, delay before start).
func (p hostRelaunchPolicy) nextRelaunchAttempt(attemptsSoFar int, uptime time.Duration) (bool, int, time.Duration) {
	attempts := attemptsSoFar
	if p.StableAfter > 0 && uptime >= p.StableAfter {
		attempts = 0
	}
	if p.MaxAttempts <= 0 {
		return false, attempts, 0
	}
	if attempts >= p.MaxAttempts {
		return false, attempts, 0
	}
	next := attempts + 1
	delay := p.Backoff[len(p.Backoff)-1]
	if next-1 < len(p.Backoff) {
		delay = p.Backoff[next-1]
	}
	return true, next, delay
}

func hostRestartingPageURL(attempt, max int, detail string) string {
	q := url.Values{}
	q.Set("attempt", strconv.Itoa(attempt))
	q.Set("max", strconv.Itoa(max))
	msg := strings.TrimSpace(detail)
	if msg == "" {
		msg = "Host exited unexpectedly"
	}
	if len(msg) > 400 {
		msg = msg[:400] + "…"
	}
	q.Set("message", msg)
	return "/shell-ui/host-restarting.html?" + q.Encode()
}

func hostRelaunchStatusLine(attempt, max int) string {
	return fmt.Sprintf("Restarting Host… (%d/%d)", attempt, max)
}
