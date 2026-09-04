import { WML } from "@wailsio/runtime";
import { ShellService } from "../bindings/github.com/xinghaix/dsh-desktop/dsh-plugin-desktop/wails";

WML.Enable();

const statusEl = document.getElementById("status")!;
const hostUrlEl = document.getElementById("host-url")! as HTMLInputElement;
const logEl = document.getElementById("log")!;

function log(message: string) {
    const line = `[${new Date().toISOString()}] ${message}`;
    logEl.textContent = `${line}\n${logEl.textContent ?? ""}`.trim();
}

async function refreshStatus() {
    try {
        statusEl.textContent = `${await ShellService.Status()} | ${await ShellService.HostSidecarStatus()}`;
        const current = await ShellService.CurrentURL();
        if (current && !hostUrlEl.value) hostUrlEl.value = current;
    } catch (err) {
        statusEl.textContent = `Shell unavailable: ${String(err)}`;
    }
}

document.getElementById("btn-sidecar")?.addEventListener("click", async () => {
    try {
        const ready = await ShellService.StartHostSidecar(hostUrlEl.value.trim());
        log(`Host ready: ${ready}`);
        await refreshStatus();
    } catch (err) {
        log(`StartHostSidecar failed: ${String(err)}`);
    }
});

document.getElementById("btn-load")!.addEventListener("click", async () => {
    const url = hostUrlEl.value.trim();
    if (!url) {
        log("Enter a Host UI URL first (http://127.0.0.1:PORT/).");
        return;
    }
    try {
        await ShellService.LoadHostURL(url);
        log(`Navigated to ${url}`);
        await refreshStatus();
    } catch (err) {
        log(`LoadHostURL failed: ${String(err)}`);
    }
});

document.getElementById("btn-control")!.addEventListener("click", async () => {
    await ShellService.ShowControlUI();
    await refreshStatus();
});

document.getElementById("btn-dir")!.addEventListener("click", async () => {
    try {
        const path = await ShellService.OpenDirectoryDialog();
        log(path ? `Selected: ${path}` : "Directory dialog cancelled");
    } catch (err) {
        log(`OpenDirectoryDialog failed: ${String(err)}`);
    }
});

document.getElementById("btn-about")!.addEventListener("click", async () => {
    await ShellService.ShowInfoDialog(
        "DSH Desktop",
        "Wails v3 native shell.\nCordis Host remains a Node sidecar during the hybrid migration.",
    );
});

document.getElementById("btn-hide")!.addEventListener("click", async () => {
    await ShellService.HideWindow();
});

document.getElementById("btn-quit")!.addEventListener("click", async () => {
    await ShellService.Quit();
});

void refreshStatus();
