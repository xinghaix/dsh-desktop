package main

import (
	"fmt"
	"runtime"
)

const desktopAppUserModelID = "ai.deepseek.dsh.desktop"

// PlatformIdentityStatus documents what the hybrid shell owns vs Electron-only.
type PlatformIdentityStatus struct {
	AppID           string `json:"appId"`
	AppUserModelID  string `json:"appUserModelId"`
	Dock            string `json:"dock"`
	CrashReporter   string `json:"crashReporter"`
	PackagedUpdates string `json:"packagedUpdates"`
	Applied         string `json:"applied"`
}

func (c *CapabilitiesService) PlatformIdentity() PlatformIdentityStatus {
	status := PlatformIdentityStatus{
		AppID:  desktopAppUserModelID,
		AppUserModelID: "windows-only; applied at shell start when GOOS=windows",
		Dock: "macOS: WebviewWindow.Flash (Dock bounce) + tray tooltip badge count; " +
			"numeric Dock badge (app.setBadgeCount) unavailable in Wails v3 beta; " +
			"MacOptions.ApplicationShouldTerminateAfterLastWindowClosed=false",
		CrashReporter: "file-based crash-evidence (active-run.json + panic-*.txt); " +
			"Electron Crashpad/minidumps permanently unavailable in Wails/Node Host",
		PackagedUpdates: "CheckForUpdates/DownloadAndInstallUpdate for macOS, Windows, and Linux AppImage",
		Applied:         "pending",
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.identityApplied != "" {
		status.Applied = c.identityApplied
	}
	return status
}

func (c *CapabilitiesService) ApplyPlatformIdentity() PlatformIdentityStatus {
	applied := applyPlatformIdentityBestEffort()
	c.mu.Lock()
	c.identityApplied = applied
	c.mu.Unlock()
	status := c.PlatformIdentity()
	status.Applied = applied
	return status
}

func applyPlatformIdentityBestEffort() string {
	switch runtime.GOOS {
	case "windows":
		if err := setWindowsAppUserModelID(desktopAppUserModelID); err != nil {
			return fmt.Sprintf("windows AppUserModelID failed: %v", err)
		}
		return "windows AppUserModelID=" + desktopAppUserModelID
	case "darwin":
		return "darwin: Flash+tray tooltip attention; numeric Dock badge blocked in Wails v3 beta"
	default:
		return "linux: tray tooltip attention; AppUserModelID N/A; file-based crash-evidence active"
	}
}
