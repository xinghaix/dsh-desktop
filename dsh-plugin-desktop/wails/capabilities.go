package main

import (
	"fmt"
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

// CapabilitiesService ports Electron DesktopRuntime system surfaces that are
// still feasible in the hybrid Wails shell: notifications, save/export dialogs,
// reveal-in-folder, terminal launch, and a lightweight update check stub.
type CapabilitiesService struct {
	mu       sync.Mutex
	app      *application.App
	shell    *ShellService
	notifier *notifications.NotificationService
	update   UpdateCheckResult
}

// UpdateCheckResult is a lightweight update probe result (not a full installer).
type UpdateCheckResult struct {
	CheckedAt   string `json:"checkedAt"`
	CurrentHint string `json:"currentHint"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
}

func NewCapabilitiesService(shell *ShellService, notifier *notifications.NotificationService) *CapabilitiesService {
	return &CapabilitiesService{shell: shell, notifier: notifier}
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
	return notifier.SendNotification(notifications.NotificationOptions{
		ID:    fmt.Sprintf("dsh-%d", time.Now().UnixNano()),
		Title: title,
		Body:  body,
	})
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
// Packaged DSH terminal shims remain Electron/macOS/Windows-owned; this is the
// hybrid fallback (xdg-terminal-exec / Terminal.app / wt.exe / cmd).
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

// CheckForUpdates performs a lightweight local probe (not electron-updater).
// Full download/install remains Electron / future wails3 updater packaging debt.
func (c *CapabilitiesService) CheckForUpdates() UpdateCheckResult {
	hint := "dev"
	if _, pluginDir, _, err := locateWailsLayout(); err == nil {
		pkg := filepath.Join(pluginDir, "package.json")
		if raw, err := os.ReadFile(pkg); err == nil {
			if v := extractJSONStringField(string(raw), "version"); v != "" {
				hint = v
			}
		}
	}
	result := UpdateCheckResult{
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		CurrentHint: hint,
		Status:      "deferred",
		Detail:      "Hybrid shell: update download/install still uses Electron desktop update adapters; Wails packaging will switch to wails3 updater later.",
	}
	c.mu.Lock()
	c.update = result
	c.mu.Unlock()
	return result
}

// LastUpdateCheck returns the most recent CheckForUpdates result.
func (c *CapabilitiesService) LastUpdateCheck() UpdateCheckResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.update
}

// LanHttpsStatus reports that LAN HTTPS remains Host/Electron-owned for now.
func (c *CapabilitiesService) LanHttpsStatus() string {
	return "lan-https=host-owned (Electron DesktopLanHttpsRuntime); Wails shell does not terminate TLS"
}

func extractJSONStringField(raw, key string) string {
	// Tiny non-general extractor good enough for package.json "version".
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
