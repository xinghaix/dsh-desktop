package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

const desktopVersionEndpoint = "https://www.dshdesktop.cn/api/desktop/version"
const desktopDownloadMac = "https://www.dshdesktop.cn/api/downloads/mac"
const desktopDownloadWin = "https://www.dshdesktop.cn/api/downloads/windows"
const desktopDownloadLinux = "https://www.dshdesktop.cn/api/downloads/linux"
const desktopCurrentVersionHeader = "X-DSH-Desktop-Version"
const desktopReleaseChannelHeader = "X-DSH-Desktop-Channel"
const desktopTargetVersionHeader = "X-DSH-Desktop-Target-Version"
const maxUpdateDownloadBytes = 1024 * 1024 * 1024

// CapabilitiesService ports Electron DesktopRuntime system surfaces that are
// still feasible in the hybrid Wails shell: notifications, save/export dialogs,
// reveal-in-folder, terminal launch, and update check/download/open.
type CapabilitiesService struct {
	mu              sync.Mutex
	app             *application.App
	shell           *ShellService
	notifier        *notifications.NotificationService
	crash           *CrashEvidenceService
	update          UpdateCheckResult
	lanHTTPS        string
	identityApplied string
	attentionCount  int
	tray            *application.SystemTray
}

// UpdateCheckResult is the hybrid update probe / download result.
type UpdateCheckResult struct {
	CheckedAt     string `json:"checkedAt"`
	CurrentHint   string `json:"currentHint"`
	LatestHint    string `json:"latestHint"`
	Status        string `json:"status"`
	Detail        string `json:"detail"`
	DownloadPath  string `json:"downloadPath,omitempty"`
	CanDownload   bool   `json:"canDownload"`
	ReleaseChannel string `json:"releaseChannel"`
}

func NewCapabilitiesService(shell *ShellService, notifier *notifications.NotificationService) *CapabilitiesService {
	return &CapabilitiesService{shell: shell, notifier: notifier, lanHTTPS: "lan-https=host-owned (awaiting Host announce)"}
}

func (c *CapabilitiesService) attach(app *application.App) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.app = app
}

// NotifyAttention shows a native notification (Electron notifyAttention stand-in).
func (c *CapabilitiesService) NotifyAttention(title, body string) error {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return fmt.Errorf("title is required")
	}
	c.mu.Lock()
	notifier := c.notifier
	c.mu.Unlock()
	if notifier == nil {
		return fmt.Errorf("notification service is not attached")
	}
	_, _ = notifier.RequestNotificationAuthorization()
	err := notifier.SendNotification(notifications.NotificationOptions{
		ID:    fmt.Sprintf("dsh-%d", time.Now().UnixNano()),
		Title: title,
		Body:  body,
	})
	_ = c.RequestUserAttention(1)
	return err
}

// SaveFileDialog opens a native save dialog and returns the chosen path.
func (c *CapabilitiesService) SaveFileDialog(defaultFilename string) (string, error) {
	c.mu.Lock()
	app := c.app
	c.mu.Unlock()
	if app == nil {
		return "", fmt.Errorf("application is not attached")
	}
	dialog := app.Dialog.SaveFile().SetMessage("Save File").CanCreateDirectories(true)
	if name := strings.TrimSpace(defaultFilename); name != "" {
		dialog = dialog.SetFilename(name)
	}
	return dialog.PromptForSingleSelection()
}

// ExportTextFile writes contents to a path chosen via the save dialog.
func (c *CapabilitiesService) ExportTextFile(defaultFilename, contents string) (string, error) {
	path, err := c.SaveFileDialog(defaultFilename)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return path, err
	}
	return path, nil
}

// RevealInFileManager opens the OS file manager at path (selectFile when possible).
func (c *CapabilitiesService) RevealInFileManager(path string, selectFile bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	c.mu.Lock()
	app := c.app
	c.mu.Unlock()
	if app == nil {
		return fmt.Errorf("application is not attached")
	}
	return app.Env.OpenFileManager(path, selectFile)
}

// OpenTerminal launches a system terminal in a useful working directory.
func (c *CapabilitiesService) OpenTerminal(workdir string) error {
	workdir = strings.TrimSpace(workdir)
	if workdir == "" {
		if wd, err := os.Getwd(); err == nil {
			workdir = wd
		} else {
			workdir = os.TempDir()
		}
	}
	if st, err := os.Stat(workdir); err != nil || !st.IsDir() {
		return fmt.Errorf("workdir must be an existing directory")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Terminal", workdir)
	case "windows":
		if _, err := exec.LookPath("wt.exe"); err == nil {
			cmd = exec.Command("wt.exe", "-d", workdir)
		} else {
			cmd = exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k", "cd", "/d", workdir)
		}
	default:
		if _, err := exec.LookPath("xdg-terminal-exec"); err == nil {
			cmd = exec.Command("xdg-terminal-exec")
			cmd.Dir = workdir
		} else if path, err := exec.LookPath("x-terminal-emulator"); err == nil {
			cmd = exec.Command(path)
			cmd.Dir = workdir
		} else if path, err := exec.LookPath("gnome-terminal"); err == nil {
			cmd = exec.Command(path, "--working-directory="+workdir)
		} else {
			return fmt.Errorf("no system terminal found (xdg-terminal-exec / x-terminal-emulator / gnome-terminal)")
		}
	}
	cmd.Env = os.Environ()
	return cmd.Start()
}

func currentPackageVersion() string {
	hint := "dev"
	if _, pluginDir, _, err := locateWailsLayout(); err == nil {
		pkg := filepath.Join(pluginDir, "package.json")
		if raw, err := os.ReadFile(pkg); err == nil {
			if v := extractJSONStringField(string(raw), "version"); v != "" {
				hint = v
			}
		}
	}
	return hint
}

func updateDownloadURL() (string, string, bool) {
	switch runtime.GOOS {
	case "darwin":
		return desktopDownloadMac, "DSH-Desktop-update.dmg", true
	case "windows":
		return desktopDownloadWin, "DSH-Desktop-Setup.exe", true
	case "linux":
		// Packaged AppImage / .deb endpoint (same API family as mac/win).
		return desktopDownloadLinux, "DSH-Desktop.AppImage", true
	default:
		return "", "", false
	}
}

// CheckForUpdates probes the public Desktop version endpoint (same as Electron checker).
func (c *CapabilitiesService) CheckForUpdates() UpdateCheckResult {
	current := currentPackageVersion()
	_, _, canDownload := updateDownloadURL()
	result := UpdateCheckResult{
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		CurrentHint:    current,
		Status:         "error",
		Detail:         "",
		CanDownload:    canDownload,
		ReleaseChannel: "stable",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, desktopVersionEndpoint, nil)
	if err != nil {
		result.Detail = err.Error()
		c.storeUpdate(result)
		return result
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set(desktopReleaseChannelHeader, "stable")
	if current != "dev" {
		req.Header.Set(desktopCurrentVersionHeader, current)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Status = "network-error"
		result.Detail = err.Error()
		c.storeUpdate(result)
		return result
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if err != nil {
		result.Detail = err.Error()
		c.storeUpdate(result)
		return result
	}
	if resp.StatusCode != http.StatusOK {
		result.Status = "http-error"
		result.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
		c.storeUpdate(result)
		return result
	}
	latest := parseVersionJSON(string(body))
	result.LatestHint = latest
	if latest == "" {
		result.Status = "invalid-response"
		result.Detail = "version endpoint returned no parseable version"
		c.storeUpdate(result)
		return result
	}
	cmp := compareLooseSemVer(latest, current)
	if current == "dev" || cmp > 0 {
		result.Status = "update-available"
		result.Detail = fmt.Sprintf("latest=%s current=%s; call DownloadAndInstallUpdate to fetch the installer (macOS/Windows/Linux).", latest, current)
	} else {
		result.Status = "up-to-date"
		result.Detail = fmt.Sprintf("current=%s is not older than latest=%s", current, latest)
	}
	if !canDownload {
		result.Detail += " Download/install is not offered on this OS."
	}
	c.storeUpdate(result)
	return result
}

// DownloadAndInstallUpdate checks for an update, downloads the installer to a
// user-chosen path, and opens it with the OS default handler (Electron handoff).
func (c *CapabilitiesService) DownloadAndInstallUpdate() UpdateCheckResult {
	check := c.CheckForUpdates()
	if check.Status != "update-available" {
		return check
	}
	downloadURL, defaultName, canDownload := updateDownloadURL()
	if !canDownload {
		check.Status = "unsupported-platform"
		check.Detail = "Installer download is not wired for this OS in the hybrid shell."
		c.storeUpdate(check)
		return check
	}
	version := check.LatestHint
	if version == "" {
		check.Status = "error"
		check.Detail = "missing latest version"
		c.storeUpdate(check)
		return check
	}
	dest, err := c.SaveFileDialog(defaultName)
	if err != nil {
		check.Status = "dialog-error"
		check.Detail = err.Error()
		c.storeUpdate(check)
		return check
	}
	if dest == "" {
		check.Status = "cancelled"
		check.Detail = "user cancelled save dialog"
		c.storeUpdate(check)
		return check
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		check.Status = "error"
		check.Detail = err.Error()
		c.storeUpdate(check)
		return check
	}
	req.Header.Set(desktopTargetVersionHeader, version)
	req.Header.Set(desktopReleaseChannelHeader, "stable")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		check.Status = "download-error"
		check.Detail = err.Error()
		c.storeUpdate(check)
		return check
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		check.Status = "download-http-error"
		check.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
		c.storeUpdate(check)
		return check
	}
	tmp := dest + ".partial"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		check.Status = "write-error"
		check.Detail = err.Error()
		c.storeUpdate(check)
		return check
	}
	written, err := io.Copy(f, io.LimitReader(resp.Body, maxUpdateDownloadBytes+1))
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		check.Status = "write-error"
		check.Detail = err.Error()
		c.storeUpdate(check)
		return check
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		check.Status = "write-error"
		check.Detail = closeErr.Error()
		c.storeUpdate(check)
		return check
	}
	if written > maxUpdateDownloadBytes {
		_ = os.Remove(tmp)
		check.Status = "too-large"
		check.Detail = "installer exceeded 1GiB limit"
		c.storeUpdate(check)
		return check
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		check.Status = "write-error"
		check.Detail = err.Error()
		c.storeUpdate(check)
		return check
	}
	check.DownloadPath = dest
	if err := openDownloadedUpdate(dest); err != nil {
		check.Status = "downloaded-open-failed"
		check.Detail = fmt.Sprintf("saved %s but failed to open: %v", dest, err)
		c.storeUpdate(check)
		return check
	}
	check.Status = "downloaded"
	check.Detail = fmt.Sprintf("saved and opened installer at %s (version %s)", dest, version)
	c.storeUpdate(check)
	_ = c.NotifyAttention("DSH Desktop update", check.Detail)
	return check
}

func openDownloadedUpdate(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd.exe", "/c", "start", "", path).Start()
	default:
		if _, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command("xdg-open", path).Start()
		}
		return fmt.Errorf("no opener for %s", runtime.GOOS)
	}
}

// LastUpdateCheck returns the most recent CheckForUpdates / Download result.
func (c *CapabilitiesService) LastUpdateCheck() UpdateCheckResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.update
}

// IngestLanHttpsAnnounceLine parses DSH_HOST_LAN_HTTPS … from Host stdout.
func (c *CapabilitiesService) IngestLanHttpsAnnounceLine(line string) bool {
	const prefix = "DSH_HOST_LAN_HTTPS "
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, prefix) {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lanHTTPS = strings.TrimSpace(strings.TrimPrefix(line, prefix))
	return true
}

// LanHttpsStatus reports Host-announced LAN HTTPS state when available.
func (c *CapabilitiesService) LanHttpsStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lanHTTPS == "" {
		return "lan-https=host-owned (Electron DesktopLanHttpsRuntime); awaiting Host announce"
	}
	return "lan-https=" + c.lanHTTPS
}


func (c *CapabilitiesService) attachCrash(crash *CrashEvidenceService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.crash = crash
}

func (c *CapabilitiesService) attachTray(tray *application.SystemTray) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tray = tray
}

// CrashEvidenceStatus exposes file-based crash evidence (Crashpad alternative).
func (c *CapabilitiesService) CrashEvidenceStatus() string {
	c.mu.Lock()
	crash := c.crash
	c.mu.Unlock()
	if crash == nil {
		return "crash-evidence=not-attached (file-based; Electron Crashpad unavailable)"
	}
	return crash.Status()
}

// RequestUserAttention flashes the dock/taskbar and updates tray badge text.
// Wails v3 beta has no Electron app.setBadgeCount; Flash() bounces the macOS
// Dock / flashes the Windows taskbar. Badge count is reflected in the tray tooltip.
func (c *CapabilitiesService) RequestUserAttention(count int) error {
	if count < 0 {
		count = 0
	}
	c.mu.Lock()
	c.attentionCount = count
	shell := c.shell
	tray := c.tray
	c.mu.Unlock()
	if shell != nil {
		shell.FlashMainWindow(count > 0)
	}
	if tray != nil {
		if count > 0 {
			tray.SetTooltip(fmt.Sprintf("DSH Desktop (%d)", count))
		} else {
			tray.SetTooltip("DSH Desktop")
		}
	}
	return nil
}

// ClearUserAttention clears dock/taskbar flash and tray badge count.
func (c *CapabilitiesService) ClearUserAttention() error {
	return c.RequestUserAttention(0)
}

// DockAttentionStatus documents maximized Wails dock/attention APIs in use.
func (c *CapabilitiesService) DockAttentionStatus() string {
	c.mu.Lock()
	count := c.attentionCount
	c.mu.Unlock()
	return fmt.Sprintf(
		"dock-attention=count=%d; api=WebviewWindow.Flash + SystemTray.SetTooltip; "+
			"macOS Dock badge number API unavailable in Wails v3 beta (no setBadgeCount); "+
			"MacOptions.ApplicationShouldTerminateAfterLastWindowClosed=false; tray lifecycle owns quit",
		count,
	)
}

func (c *CapabilitiesService) storeUpdate(result UpdateCheckResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.update = result
}

func extractJSONStringField(raw, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(raw, needle)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(needle):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func parseVersionJSON(raw string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		// try plain string body
		s := strings.TrimSpace(raw)
		s = strings.Trim(s, `"`)
		if s != "" && strings.Contains(s, ".") {
			return strings.TrimPrefix(s, "v")
		}
		return ""
	}
	for _, key := range []string{"version", "latestVersion", "latest", "desktopVersion"} {
		if v, ok := payload[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimPrefix(strings.TrimSpace(s), "v")
			}
		}
	}
	return ""
}

// compareLooseSemVer returns >0 if a>b, <0 if a<b, 0 if equal/unparseable equal.
func compareLooseSemVer(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		var ai, bi int
		if i < len(pa) {
			fmt.Sscanf(strings.Split(pa[i], "-")[0], "%d", &ai)
		}
		if i < len(pb) {
			fmt.Sscanf(strings.Split(pb[i], "-")[0], "%d", &bi)
		}
		if ai != bi {
			return ai - bi
		}
	}
	return 0
}
