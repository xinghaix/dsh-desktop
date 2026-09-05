package main

import (
	"fmt"
	"os"
	"path/filepath"
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


func defaultHostBootstrap() (command string, urlFile string, err error) {
	_, pluginDir, repoDir, err := locateWailsLayout()
	if err != nil {
		return "", "", err
	}
	urlFile = defaultHostURLFile()
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
