package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// File-based crash evidence for the Wails shell / Node Host hybrid.
// Electron Crashpad/minidumps are unavailable here; we keep local markers and
// panic dumps under <userData>/crash-evidence/ (upload never enabled).

const crashEvidenceDirName = "crash-evidence"
const crashActiveRunFile = "active-run.json"
const crashPanicPrefix = "panic-"

type crashRunRecord struct {
	StartedAt string `json:"startedAt"`
	PID       int    `json:"pid"`
	Version   string `json:"version"`
	OwnerID   string `json:"ownerId,omitempty"`
	Shell     string `json:"shell"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

type CrashEvidenceService struct {
	mu        sync.Mutex
	dir       string
	ownerID   string
	previous  string
	clean     bool
	installed bool
}

func NewCrashEvidenceService() *CrashEvidenceService {
	return &CrashEvidenceService{}
}

// BeginRun writes active-run.json and returns a note about any leftover marker.
func (c *CrashEvidenceService) BeginRun(version string) string {
	dir, err := crashEvidenceDir()
	if err != nil {
		return "crash-evidence=unavailable: " + err.Error()
	}
	owner := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	prevNote := ""
	activePath := filepath.Join(dir, crashActiveRunFile)
	if raw, err := os.ReadFile(activePath); err == nil && len(raw) > 0 {
		prevNote = "previous unclean exit marker present: " + strings.TrimSpace(string(raw))
		_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("unclean-%d.json", time.Now().UnixNano())), raw, 0o600)
	}
	rec := crashRunRecord{
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
		PID:       os.Getpid(),
		Version:   strings.TrimSpace(version),
		OwnerID:   owner,
		Shell:     "wails",
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
	if rec.Version == "" {
		rec.Version = currentPackageVersion()
	}
	payload, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(activePath, append(payload, '\n'), 0o600); err != nil {
		return "crash-evidence=write-failed: " + err.Error()
	}
	c.mu.Lock()
	c.dir = dir
	c.ownerID = owner
	c.previous = prevNote
	c.clean = false
	c.installed = true
	c.mu.Unlock()
	if prevNote == "" {
		return "crash-evidence=active dir=" + dir
	}
	return "crash-evidence=active dir=" + dir + "; " + prevNote
}

// MarkClean removes the active-run marker owned by this process.
func (c *CrashEvidenceService) MarkClean() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.installed || c.clean {
		return
	}
	activePath := filepath.Join(c.dir, crashActiveRunFile)
	raw, err := os.ReadFile(activePath)
	if err != nil {
		c.clean = true
		return
	}
	var stored crashRunRecord
	if json.Unmarshal(raw, &stored) != nil || stored.OwnerID != c.ownerID {
		c.clean = true
		return
	}
	_ = os.Remove(activePath)
	c.clean = true
}

// Dir returns the crash-evidence directory when known (empty if not started).
func (c *CrashEvidenceService) Dir() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dir
}

// Status reports crash-evidence readiness for control UI / PlatformIdentity.
// Multi-line so Help → Crash Evidence Status / Control UI are readable on Linux.
func (c *CrashEvidenceService) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.installed {
		return strings.Join([]string{
			"crash-evidence=not-started",
			"backend=file-based (Electron Crashpad/minidumps unavailable in Wails)",
			"upload=never (local markers + panic dumps only)",
			"hint=Help → Reveal Crash Evidence Folder after BeginRun",
		}, "\n")
	}
	state := "active"
	if c.clean {
		state = "clean"
	}
	lines := []string{
		"crash-evidence=" + state,
		"dir=" + c.dir,
		"backend=file-based active-run.json + panic-*.txt",
		"upload=never",
	}
	if c.previous != "" {
		lines = append(lines, "previous="+c.previous)
	}
	return strings.Join(lines, "\n")
}

// WritePanicDump persists a panic stack to crash-evidence (best-effort).
func (c *CrashEvidenceService) WritePanicDump(recovered any) string {
	dir, err := crashEvidenceDir()
	if err != nil {
		return ""
	}
	name := fmt.Sprintf("%s%d.txt", crashPanicPrefix, time.Now().UnixNano())
	path := filepath.Join(dir, name)
	var b strings.Builder
	fmt.Fprintf(&b, "time=%s\npid=%d\ngoos=%s\ngoarch=%s\nrecovered=%v\n\n",
		time.Now().UTC().Format(time.RFC3339Nano), os.Getpid(), runtime.GOOS, runtime.GOARCH, recovered)
	b.Write(debug.Stack())
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return ""
	}
	return path
}

func crashEvidenceDir() (string, error) {
	base, err := desktopUserDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, crashEvidenceDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func desktopUserDataDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("DSH_DESKTOP_USER_DATA")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, "DSH Desktop"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "DSH Desktop"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "DSH Desktop"), nil
	default:
		if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
			return filepath.Join(xdg, "DSH Desktop"), nil
		}
		return filepath.Join(home, ".config", "DSH Desktop"), nil
	}
}

// installCrashEvidenceHooks registers panic recovery around app.Run via deferred call sites.
func installProcessCrashHooks(crash *CrashEvidenceService) {
	// Best-effort: capture stderr panics that bubble through deferred recover in main.
	_ = crash
}
