/**
 * Maps Electron native-ui custom schemes (dsh-recovery:, dsh-setup-wizard:,
 * dsh-profile-selector:, dsh-profile-create:) onto Wails AuxWindowService
 * bindings so Vite-built React native-ui pages can run inside aux windows.
 */
(function installDshSchemeBridge() {
  if (window.__DSH_SCHEME_BRIDGE__) return;
  const call = async (name, ...args) => {
    const wails = await import("/wails/runtime.js");
    return wails.Call.ByName(name, ...args);
  };

  async function handleScheme(href) {
    let url;
    try { url = new URL(href); } catch { return false; }
    const protocol = url.protocol;
    const action = (url.hostname || url.pathname.replace(/^\//, "") || "").toLowerCase();
    const name = url.searchParams.get("name") || "";

    if (protocol === "dsh-recovery:") {
      const map = {
        restart: "restart",
        "safe-mode": "safe-mode",
        safemode: "safe-mode",
        "enter-safe-mode": "safe-mode",
        quit: "quit",
        profiles: "profiles",
        control: "control",
        "preview-checkpoint": "preview-checkpoint",
        "open-checkpoint": "open-checkpoint",
        "confirm-checkpoint": "confirm-checkpoint",
        "rollback-checkpoint": "rollback-checkpoint",
        "preview-uninstall": "preview-uninstall",
        "confirm-uninstall": "confirm-uninstall",
        "uninstall-plugin": "uninstall-plugin",
        "export-diagnostics": "export-diagnostics",
        "show-diagnostics": "show-diagnostics",
        "save-diagnostics": "save-diagnostics",
        "open-settings-document": "open-settings-document",
        "open-profile-patch": "open-profile-patch",
        "open-profile-manifest": "open-profile-manifest",
        "open-profile-directory": "open-profile-directory",
        "open-terminal": "open-terminal",
        "open-profile-creator": "open-profile-creator",
        "switch-profile": "switch-profile",
      };
      const mapped = map[action];
      if (!mapped) return false;
      await call("main.AuxWindowService.CompleteRecovery", mapped);
      return true;
    }
    if (protocol === "dsh-setup-wizard:") {
      if (action === "complete" || action === "continue") {
        await call("main.AuxWindowService.CompleteSetup", "continue");
        return true;
      }
      if (action === "skip" || action === "quit") {
        await call("main.AuxWindowService.CompleteSetup", "quit");
        return true;
      }
      return false;
    }
    if (protocol === "dsh-profile-selector:") {
      const map = { cancel: "cancel", create: "create", restart: "restart", switch: "switch", select: "switch" };
      const mapped = map[action];
      if (!mapped) return false;
      await call("main.AuxWindowService.CompleteProfileSelection", mapped, name);
      return true;
    }
    if (protocol === "dsh-profile-create:") {
      if (action === "cancel") {
        await call("main.AuxWindowService.CompleteProfileCreate", "cancel", "");
        return true;
      }
      if (action === "submit" || action === "create") {
        await call("main.AuxWindowService.CompleteProfileCreate", "create", name || url.searchParams.get("profile") || "");
        return true;
      }
      return false;
    }
    return false;
  }

  const assign = window.location.assign.bind(window.location);
  window.location.assign = function patchedAssign(url) {
    const href = String(url);
    if (/^dsh-[\w-]+:/i.test(href)) {
      handleScheme(href).catch((err) => console.error("dsh scheme bridge", err));
      return;
    }
    return assign(url);
  };

  document.addEventListener("click", (ev) => {
    const a = ev.target && ev.target.closest ? ev.target.closest("a[href]") : null;
    if (!a) return;
    const href = a.getAttribute("href") || "";
    if (/^dsh-[\w-]+:/i.test(href)) {
      ev.preventDefault();
      handleScheme(href).catch((err) => console.error("dsh scheme bridge", err));
    }
  }, true);

  window.__DSH_SCHEME_BRIDGE__ = { handleScheme };
})();
