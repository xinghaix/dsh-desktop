package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// nativeUIDocument resolves Vite React native-ui under frontend/dist/aux/native-ui only.
// Falls back to empty so resolveAuxURL uses simplified /aux HTML (never file://).
func nativeUIDocument(name string) string {
	_, pluginDir, _, err := locateWailsLayout()
	if err != nil {
		return ""
	}
	// Prefer frontend/dist/aux/native-ui (asset-server / go:embed). Never use lib/ via file://.
	candidates := []string{
		filepath.Join(pluginDir, "wails", "frontend", "dist", "aux", "native-ui", name+".html"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func recoveryNativeState(detail string, profiles []string) string {
	payload := map[string]any{
		"failureStage":      "host-sidecar",
		"failureDetail":     detail,
		"requested":         false,
		"restartReady":      true,
		"terminalAvailable": false,
		"safeModeActive":    false,
		"busy":              false,
		"activeTab":         "quick",
		"locale":            "en",
		"profiles":          profiles,
		"notice":            nil,
		"wailsHybrid":       true,
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
		"locale":   "en",
		"profiles": items,
		"token":    fmt.Sprintf("wails-%d", time.Now().UnixNano()),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
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
      const map = {restart:'restart','safe-mode':'safe-mode',safemode:'safe-mode',quit:'quit',profiles:'profiles',control:'control'};
      const mapped = map[action]; if (!mapped) return false;
      await call('main.AuxWindowService.CompleteRecovery', mapped); return true;
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
