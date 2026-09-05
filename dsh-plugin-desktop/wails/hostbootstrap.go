package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// hostAutostartEnabled is true unless DSH_HOST_AUTOSTART is 0/false/off/no.
func hostAutostartEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("DSH_HOST_AUTOSTART")))
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func defaultHostURLFile() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("dsh-host-url-%d", os.Getpid()))
}

func baseName(path string) string {
	return filepath.Base(filepath.Clean(path))
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'"'"'`) + "'"
}

// locateWailsLayout finds wails/, dsh-plugin-desktop/, and monorepo root.
func locateWailsLayout() (wailsDir, pluginDir, repoDir string, err error) {
	var candidates []string
	if executablePath, execErr := os.Executable(); execErr == nil {
		candidates = append(candidates, filepath.Dir(executablePath))
	}
	if wd, e := os.Getwd(); e == nil {
		candidates = append(candidates, wd)
	}
	for _, start := range candidates {
		dir := start
		for i := 0; i < 8; i++ {
			if baseName(dir) == "wails" && fileExists(filepath.Join(dir, "hostsidecar.go")) {
				return dir, filepath.Dir(dir), filepath.Dir(filepath.Dir(dir)), nil
			}
			probe := filepath.Join(dir, "wails", "hostsidecar.go")
			if fileExists(probe) {
				return filepath.Join(dir, "wails"), dir, filepath.Dir(dir), nil
			}
			probe = filepath.Join(dir, "dsh-plugin-desktop", "wails", "hostsidecar.go")
			if fileExists(probe) {
				return filepath.Join(dir, "dsh-plugin-desktop", "wails"), filepath.Join(dir, "dsh-plugin-desktop"), dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", "", "", fmt.Errorf("could not locate dsh-plugin-desktop/wails layout from cwd/executable")
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func envTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envFalsy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func locateElectronExecutable(repoDir, pluginDir string) string {
	if p := strings.TrimSpace(os.Getenv("ELECTRON_PATH")); p != "" && fileExists(p) {
		return p
	}
	var candidates []string
	for _, root := range []string{repoDir, pluginDir} {
		base := filepath.Join(root, "node_modules", "electron", "dist")
		switch runtime.GOOS {
		case "darwin":
			candidates = append(candidates, filepath.Join(base, "Electron.app", "Contents", "MacOS", "Electron"))
		case "windows":
			candidates = append(candidates, filepath.Join(base, "electron.exe"))
		default:
			candidates = append(candidates, filepath.Join(base, "electron"))
		}
		pathTxt := filepath.Join(root, "node_modules", "electron", "path.txt")
		if fileExists(pathTxt) {
			if raw, err := os.ReadFile(pathTxt); err == nil {
				p := strings.TrimSpace(string(raw))
				if p != "" {
					if !filepath.IsAbs(p) {
						p = filepath.Join(root, "node_modules", "electron", p)
					}
					candidates = append(candidates, p)
				}
			}
		}
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func profileNodeModulesPresent() bool {
	home, _ := os.UserHomeDir()
	roots := []string{}
	if home != "" {
		switch runtime.GOOS {
		case "darwin":
			roots = append(roots, filepath.Join(home, "Library", "Application Support", "DSH Desktop", "profiles"))
		case "windows":
			if appdata := os.Getenv("APPDATA"); appdata != "" {
				roots = append(roots, filepath.Join(appdata, "DSH Desktop", "profiles"))
			}
		default:
			config := os.Getenv("XDG_CONFIG_HOME")
			if config == "" {
				config = filepath.Join(home, ".config")
			}
			roots = append(roots, filepath.Join(config, "DSH Desktop", "profiles"))
		}
	}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && dirExists(filepath.Join(root, e.Name(), "node_modules")) {
				return true
			}
		}
	}
	return false
}

type hostLauncherMode string

const (
	hostLauncherNode           hostLauncherMode = "node"
	hostLauncherElectronAsNode hostLauncherMode = "electron-as-node"
	hostLauncherElectronMain   hostLauncherMode = "electron-main"
)

func resolveHostLauncherModeGo(hostMainExists bool, electronPath string) (hostLauncherMode, string) {
	launcher := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(os.Getenv("DSH_HOST_LAUNCHER"), "_", "-")))
	switch launcher {
	case "node":
		return hostLauncherNode, "DSH_HOST_LAUNCHER=node"
	case "electron-as-node", "electronasnode", "run-as-node":
		return hostLauncherElectronAsNode, "DSH_HOST_LAUNCHER=electron-as-node"
	case "electron-main", "electron", "main":
		return hostLauncherElectronMain, "DSH_HOST_LAUNCHER=electron-main"
	}
	if envTruthy("DSH_HOST_ELECTRON_AS_NODE") {
		return hostLauncherElectronAsNode, "DSH_HOST_ELECTRON_AS_NODE=1"
	}
	if envFalsy("DSH_HOST_ELECTRON_AS_NODE") && hostMainExists {
		return hostLauncherNode, "DSH_HOST_ELECTRON_AS_NODE=0"
	}
	if !hostMainExists {
		return hostLauncherElectronMain, "lib/host-main.js missing"
	}
	if profileNodeModulesPresent() {
		if electronPath != "" {
			return hostLauncherElectronAsNode, "profile node_modules present; prefer ELECTRON_RUN_AS_NODE"
		}
		return hostLauncherNode, "profile node_modules present but Electron binary missing; stock Node"
	}
	return hostLauncherNode, "no profile node_modules detected; stock Node Host"
}

// hostSidecarArgument is the Cordis Host flag that enables Wails sidecar announce lines.
const hostSidecarArgument = "--dsh-wails-host-sidecar"

// commandFromHostBin builds a shell command that launches a user-provided Host binary
// or Node entry in Wails sidecar mode.
//
// DSH_BIN may point at:
//   - a JS Host entry (*.js / *.mjs / *.cjs), run with node
//   - an executable shim such as dsh-desktop / dsh-plugin-desktop
//
// Bare dsh (@deepseek-ai/dsh CLI) is intentionally not accepted here: Desktop Host
// requires the Cordis desktop plugin stack (AuthProxy announce, Recovery RPC, etc.).
func commandFromHostBin(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return "", fmt.Errorf("empty Host bin")
	}
	if strings.ContainsAny(bin, "\x00\n\r") {
		return "", fmt.Errorf("Host bin must not contain NUL or newlines")
	}
	lower := strings.ToLower(bin)
	switch {
	case strings.HasSuffix(lower, ".js"), strings.HasSuffix(lower, ".mjs"), strings.HasSuffix(lower, ".cjs"):
		if !fileExists(bin) {
			return "", fmt.Errorf("DSH_BIN JS entry not found: %s", bin)
		}
		return fmt.Sprintf("export DSH_WAILS_HOST_SIDECAR=1; node %s %s", shellQuote(bin), hostSidecarArgument), nil
	default:
		resolved := bin
		if !filepath.IsAbs(bin) {
			if p, err := exec.LookPath(bin); err == nil {
				resolved = p
			}
		}
		if filepath.IsAbs(resolved) && !fileExists(resolved) {
			return "", fmt.Errorf("DSH_BIN executable not found: %s", resolved)
		}
		return fmt.Sprintf("export DSH_WAILS_HOST_SIDECAR=1; %s %s", shellQuote(resolved), hostSidecarArgument), nil
	}
}

// discoverUserInstalledHostCommand finds a user-installed Desktop Host when the
// monorepo/plugin layout is unavailable (e.g. AppImage shell-only package).
//
// Order:
//  1. DSH_BIN (explicit override; always checked by defaultHostBootstrap first)
//  2. PATH executables: dsh-desktop, then dsh-plugin-desktop
//
// Returns ok=false when nothing usable is found.
func discoverUserInstalledHostCommand() (command string, reason string, ok bool) {
	if bin := strings.TrimSpace(os.Getenv("DSH_BIN")); bin != "" {
		cmd, err := commandFromHostBin(bin)
		if err != nil {
			return "", "", false
		}
		return cmd, "DSH_BIN", true
	}
	for _, name := range []string{"dsh-desktop", "dsh-plugin-desktop"} {
		p, err := exec.LookPath(name)
		if err != nil || p == "" {
			continue
		}
		cmd, err := commandFromHostBin(p)
		if err != nil {
			continue
		}
		return cmd, "PATH:" + name, true
	}
	return "", "", false
}

func defaultHostBootstrap() (command string, urlFile string, err error) {
	urlFile = defaultHostURLFile()
	if bin := strings.TrimSpace(os.Getenv("DSH_BIN")); bin != "" {
		cmd, binErr := commandFromHostBin(bin)
		if binErr != nil {
			return "", "", fmt.Errorf("DSH_BIN: %w", binErr)
		}
		return cmd, urlFile, nil
	}
	_, pluginDir, repoDir, layoutErr := locateWailsLayout()
	if layoutErr != nil {
		if cmd, _, ok := discoverUserInstalledHostCommand(); ok {
			return cmd, urlFile, nil
		}
		return "", "", fmt.Errorf("no Cordis Host launcher: %v (set DSH_BIN / DSH_HOST_COMMAND / DSH_HOST_URL, or install dsh-desktop on PATH)", layoutErr)
	}
	hostMainJS := filepath.Join(pluginDir, "lib", "host-main.js")
	electronPath := locateElectronExecutable(repoDir, pluginDir)
	mode, _ := resolveHostLauncherModeGo(fileExists(hostMainJS), electronPath)
	if mode == hostLauncherElectronAsNode && fileExists(hostMainJS) && electronPath != "" {
		command = fmt.Sprintf("cd %s && export ELECTRON_RUN_AS_NODE=1; export DSH_WAILS_HOST_SIDECAR=1; %s %s %s", shellQuote(pluginDir), shellQuote(electronPath), shellQuote(hostMainJS), hostSidecarArgument)
		return command, urlFile, nil
	}
	if fileExists(hostMainJS) && mode != hostLauncherElectronMain {
		parts := []string{"node", shellQuote(hostMainJS), hostSidecarArgument}
		command = fmt.Sprintf("cd %s && export DSH_WAILS_HOST_SIDECAR=1; %s", shellQuote(pluginDir), strings.Join(parts, " "))
		return command, urlFile, nil
	}
	yarnLock := filepath.Join(repoDir, "yarn.lock")
	workspacePkg := filepath.Join(repoDir, "package.json")
	binJS := filepath.Join(pluginDir, "lib", "bin.js")
	if fileExists(yarnLock) && fileExists(workspacePkg) {
		parts := []string{"yarn", "workspace", "dsh-plugin-desktop", "start", "--", hostSidecarArgument}
		command = fmt.Sprintf("cd %s && export DSH_WAILS_HOST_SIDECAR=1; %s", shellQuote(repoDir), strings.Join(parts, " "))
		return command, urlFile, nil
	}
	if fileExists(binJS) {
		parts := []string{"node", shellQuote(binJS), hostSidecarArgument}
		command = fmt.Sprintf("cd %s && export DSH_WAILS_HOST_SIDECAR=1; %s", shellQuote(pluginDir), strings.Join(parts, " "))
		return command, urlFile, nil
	}
	if cmd, _, ok := discoverUserInstalledHostCommand(); ok {
		return cmd, urlFile, nil
	}
	return "", "", fmt.Errorf("no default Cordis Host launcher: need %s or %s (or set DSH_BIN / DSH_HOST_COMMAND / DSH_HOST_URL)", yarnLock, binJS)
}
