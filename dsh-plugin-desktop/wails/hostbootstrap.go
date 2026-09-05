package main

import (
	"fmt"
	"os"
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

func defaultHostBootstrap() (command string, urlFile string, err error) {
	_, pluginDir, repoDir, err := locateWailsLayout()
	if err != nil {
		return "", "", err
	}
	urlFile = defaultHostURLFile()
	hostMainJS := filepath.Join(pluginDir, "lib", "host-main.js")
	electronPath := locateElectronExecutable(repoDir, pluginDir)
	mode, _ := resolveHostLauncherModeGo(fileExists(hostMainJS), electronPath)
	if mode == hostLauncherElectronAsNode && fileExists(hostMainJS) && electronPath != "" {
		command = fmt.Sprintf("cd %s && export ELECTRON_RUN_AS_NODE=1; export DSH_WAILS_HOST_SIDECAR=1; %s %s --dsh-wails-host-sidecar", shellQuote(pluginDir), shellQuote(electronPath), shellQuote(hostMainJS))
		return command, urlFile, nil
	}
	if fileExists(hostMainJS) && mode != hostLauncherElectronMain {
		parts := []string{"node", shellQuote(hostMainJS), "--dsh-wails-host-sidecar"}
		command = fmt.Sprintf("cd %s && export DSH_WAILS_HOST_SIDECAR=1; %s", shellQuote(pluginDir), strings.Join(parts, " "))
		return command, urlFile, nil
	}
	yarnLock := filepath.Join(repoDir, "yarn.lock")
	workspacePkg := filepath.Join(repoDir, "package.json")
	binJS := filepath.Join(pluginDir, "lib", "bin.js")
	sidecarFlag := "--dsh-wails-host-sidecar"
	if fileExists(yarnLock) && fileExists(workspacePkg) {
		parts := []string{"yarn", "workspace", "dsh-plugin-desktop", "start", "--", sidecarFlag}
		command = fmt.Sprintf("cd %s && export DSH_WAILS_HOST_SIDECAR=1; %s", shellQuote(repoDir), strings.Join(parts, " "))
		return command, urlFile, nil
	}
	if fileExists(binJS) {
		parts := []string{"node", shellQuote(binJS), sidecarFlag}
		command = fmt.Sprintf("cd %s && export DSH_WAILS_HOST_SIDECAR=1; %s", shellQuote(pluginDir), strings.Join(parts, " "))
		return command, urlFile, nil
	}
	return "", "", fmt.Errorf("no default Cordis Host launcher: need %s or %s (or set DSH_HOST_COMMAND / DSH_HOST_URL)", yarnLock, binJS)
}
