package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// shellUIDir is the go:embed-friendly name for auxiliary HTML.
// NOTE: cannot be named "aux" — Go rejects embedding Windows reserved device
// names (AUX/CON/PRN/NUL/COM*/LPT*), which silently dropped the whole tree
// from //go:embed and left Setup/Profile/Recovery windows blank.
const shellUIDir = "shell-ui"

// nativeUIDocument resolves Vite React native-ui under frontend/dist/shell-ui/native-ui only.
// Falls back to empty so resolveAuxURL uses simplified /shell-ui HTML (never file://).
func nativeUIDocument(name string) string {
	_, pluginDir, _, err := locateWailsLayout()
	if err != nil {
		return ""
	}
	// Prefer frontend/dist/shell-ui/native-ui (asset-server / go:embed). Never use lib/ via file://.
	candidates := []string{
		filepath.Join(pluginDir, "wails", "frontend", "dist", shellUIDir, "native-ui", name+".html"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func recoveryNativeState(detail string, profiles []string, snapshot *RecoverySnapshot) string {
	profileItems := make([]map[string]any, 0, len(profiles))
	for i, name := range profiles {
		profileItems = append(profileItems, map[string]any{
			"name":       name,
			"current":    i == 0,
			"selectable": true,
		})
	}
	if detail == "" {
		detail = "Cordis Host failed to start or announce a UI URL."
	}
	payload := map[string]any{
		"locale":                  "en",
		"failureStage":            "host-boot",
		"failureDetail":           detail,
		"requested":               false,
		"diagnostics":             map[string]any{"status": "failed"},
		"busy":                    false,
		"restartReady":            true,
		"activeTab":               "quick",
		"configurationAvailable":  false,
		"terminalAvailable":       false,
		"safeModeAvailable":       true,
		"safeModeActive":          false,
		"profileCreatorAvailable": true,
		"profiles":                profileItems,
		// Omit notice: JSON null crashes RecoveryNoticeSurface (useEffect deps
		// read notice.body while only undefined is treated as "no toast").
		"wailsHybrid": true,
	}
	if snapshot != nil {
		payload["snapshot"] = snapshot
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func profileSelectorNativeState(profiles []string) string {
	items := make([]map[string]any, 0, len(profiles))
	for i, name := range profiles {
		items = append(items, map[string]any{
			"name":       name,
			"current":    i == 0,
			"selectable": true,
		})
	}
	payload := map[string]any{
		"locale":       "en",
		"profiles":     items,
		"busy":         false,
		"restartReady": true,
		"token":        fmt.Sprintf("wails-%d", time.Now().UnixNano()),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// setupWizardNativeState builds the exact base64url state Desktop Setup Wizard decodes.
// Query must be exactly locale/state/platform/frame (no extra keys such as wails=1).
func setupWizardNativeState() (state string, platform string) {
	platform = "linux"
	switch runtime.GOOS {
	case "darwin":
		platform = "darwin"
	case "windows":
		platform = "win32"
	}
	payload := map[string]any{
		"mode":            "compatibility",
		"macosMaterial":   "off",
		"windowsMaterial": "off",
		"openBrowser":     false,
		"networkExposure": "loopback",
		"market":          "disabled",
		"notifications": map[string]any{
			"enabled":                true,
			"notifyOnTurnCompletion": true,
			"notifyOnTurnFailure":    true,
			"notifyOnJobCompletion":  true,
			"notifyOnJobFailure":     true,
		},
		"profileName":   "default",
		"platform":      platform,
		"micaSupported": false,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", platform
	}
	return base64.RawURLEncoding.EncodeToString(raw), platform
}

const schemeBridgeJS = `
(function(){
  if (window.__DSH_SCHEME_BRIDGE__) return;
  const call = async (name, ...args) => {
    if (window.__dshCall) return window.__dshCall(name, ...args);
    const wails = await import("/wails/runtime.js");
    return wails.Call.ByName(name, ...args);
  };
  async function handleScheme(href) {
    let url; try { url = new URL(href); } catch { return false; }
    const protocol = url.protocol;
    const action = (url.hostname || url.pathname.replace(/^\//,'') || '').toLowerCase();
    const name = url.searchParams.get('name') || '';
    if (protocol === 'dsh-recovery:') {
      const map = {
        restart:'restart','safe-mode':'safe-mode',safemode:'safe-mode','enter-safe-mode':'safe-mode',
        quit:'quit',profiles:'profiles',control:'control',
        'preview-checkpoint':'preview-checkpoint','open-checkpoint':'open-checkpoint',
        'confirm-checkpoint':'confirm-checkpoint','rollback-checkpoint':'rollback-checkpoint',
        'preview-uninstall':'preview-uninstall','confirm-uninstall':'confirm-uninstall',
        'uninstall-plugin':'uninstall-plugin',
        'export-diagnostics':'export-diagnostics','show-diagnostics':'show-diagnostics',
        'save-diagnostics':'save-diagnostics',
        'open-settings-document':'open-settings-document','open-profile-patch':'open-profile-patch',
        'open-profile-manifest':'open-profile-manifest','open-profile-directory':'open-profile-directory',
        'open-terminal':'open-terminal','open-profile-creator':'open-profile-creator',
        'switch-profile':'switch-profile'
      };
      const mapped = map[action]; if (!mapped) return false;
      const id = url.searchParams.get('id') || '';
      await call('main.AuxWindowService.CompleteRecovery', mapped, id); return true;
    }
    if (protocol === 'dsh-setup-wizard:') {
      if (action === 'complete' || action === 'continue') { await call('main.AuxWindowService.CompleteSetup', 'continue'); return true; }
      if (action === 'skip' || action === 'quit') { await call('main.AuxWindowService.CompleteSetup', 'quit'); return true; }
      return false;
    }
    if (protocol === 'dsh-profile-selector:') {
      const map = {cancel:'cancel',create:'create',restart:'restart',switch:'switch',select:'switch'};
      const mapped = map[action]; if (!mapped) return false;
      await call('main.AuxWindowService.CompleteProfileSelection', mapped, name); return true;
    }
    if (protocol === 'dsh-profile-create:') {
      if (action === 'cancel') { await call('main.AuxWindowService.CompleteProfileCreate', 'cancel', ''); return true; }
      if (action === 'submit' || action === 'create') {
        await call('main.AuxWindowService.CompleteProfileCreate', 'create', name || url.searchParams.get('profile') || '');
        return true;
      }
    }
    return false;
  }
  const assign = window.location.assign.bind(window.location);
  window.location.assign = function(url) {
    const href = String(url);
    if (/^dsh-[\\w-]+:/i.test(href)) { handleScheme(href); return; }
    return assign(url);
  };
  window.__DSH_SCHEME_BRIDGE__ = { handleScheme };
})();
`
