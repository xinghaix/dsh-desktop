/**
 * Hybrid preload stand-in for the Wails control UI.
 * Electron exposed __DSH_DESKTOP_FILE_PATH__ via contextBridge; Host remote pages
 * do not receive this object (asset-server only). Use BridgeService bindings instead.
 */
(async function installDshBridge() {
  try {
    const wails = await import("/wails/runtime.js");
    const call = (name, ...args) => wails.Call.ByName(name, ...args);
    const bridge = {
      getPathForFile(_file) {
        throw new Error(
          "GetPathForFile unavailable in Wails hybrid shell; use OpenDirectoryDialog/OpenFileDialog",
        );
      },
      openExternal(url) {
        return call("main.BridgeService.OpenExternal", url);
      },
      authenticateHostSession(authenticationUrl, uiUrl) {
        return call("main.BridgeService.AuthenticateHostSession", authenticationUrl, uiUrl || "");
      },
      status() {
        return call("main.BridgeService.BridgeStatus");
      },
    };
    window.__DSH_DESKTOP_FILE_PATH__ = {
      getPathForFile: bridge.getPathForFile,
    };
    window.__DSH_WAILS_SHELL__ = bridge;
  } catch (err) {
    console.warn("dsh wails bridge install failed", err);
  }
})();
