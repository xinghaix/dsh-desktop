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

// commandFromHostBin builds a shell command that launches a Host binary
// or Node entry in Wails sidecar mode.
//
// Conceptually Host is the Cordis Web UI stack (same family as `dsh --profile web`).
// For Wails sidecar spawn we still need a desktop-capable entry that speaks
// announce / desktop plugins — typically host-main.js or dsh-desktop — not a
// bare `dsh` argv without the desktop sidecar contract.
//
// DSH_BIN may point at:
//   - a JS Host entry (*.js / *.mjs / *.cjs), run with node
//   - an executable shim such as dsh-desktop / dsh-plugin-desktop
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
		slash := filepath.ToSlash(bin)
		if strings.Contains(slash, "/@deepseek-ai/dsh/") {
			profile := resolveBootstrapProfile()
			// dsh CLI 使用环境变量识别桌面 sidecar，不接受 Wails 专用参数；
			// --no-open 防止 Host 另外打开系统浏览器，--port 0 让系统分配空闲端口，
			// 最终 URL 从 stdout 获取。
			return fmt.Sprintf("export DSH_WAILS_HOST_SIDECAR=1; node %s --profile %s --no-open --port 0",
				shellQuote(bin), shellQuote(profile)), nil
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
		base := strings.ToLower(baseName(resolved))
		if base == "dsh" || base == "dsh.exe" {
			profile := resolveBootstrapProfile()
			return fmt.Sprintf("export DSH_WAILS_HOST_SIDECAR=1; %s --profile %s %s",
				shellQuote(resolved), shellQuote(profile), hostSidecarArgument), nil
		}
		return fmt.Sprintf("export DSH_WAILS_HOST_SIDECAR=1; %s %s", shellQuote(resolved), hostSidecarArgument), nil
	}
}

// resolveBootstrapProfile prefers an explicit / preferred profile, else an existing
// user profile, else the Cordis `web` profile (`dsh --profile web`).
func resolveBootstrapProfile() string {
	if p := strings.TrimSpace(os.Getenv("DSH_DESKTOP_DEFAULT_PROFILE")); p != "" {
		return resolveActiveProfileName(p)
	}
	if p := strings.TrimSpace(os.Getenv("DSH_PROFILE")); p != "" {
		return resolveActiveProfileName(p)
	}
	return resolveActiveProfileName("")
}

// hostDiscoverHit is a successful Desktop Host resolution.
type hostDiscoverHit struct {
	Command string
	Reason  string
	Path    string
}

// hostDiscoverReport summarizes user-home Host probe for UX / logs (product default; packaged is opt-in).
type hostDiscoverReport struct {
	Checked []string
	Hit     *hostDiscoverHit
	Message string
}

func appendUnique(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, v := range dst {
		seen[v] = struct{}{}
	}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		dst = append(dst, v)
	}
	return dst
}

func userHomeDir() string {
	if h := strings.TrimSpace(os.Getenv("HOME")); h != "" {
		return h
	}
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

// userChosenDshHomeFile is where the shell persists a user-picked install directory.
func userChosenDshHomeFile() string {
	if ud := resolveDesktopUserDataDir(); ud != "" {
		return filepath.Join(ud, "user-dsh-home")
	}
	return ""
}

// loadUserChosenDshHome returns a persisted user-selected DSH_HOME directory, if any.
func loadUserChosenDshHome() string {
	p := userChosenDshHomeFile()
	if p == "" {
		return ""
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// saveUserChosenDshHome persists a user-selected install directory for later boots.
func saveUserChosenDshHome(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}
	p := userChosenDshHomeFile()
	if p == "" {
		return fmt.Errorf("cannot resolve Desktop userData for user-dsh-home")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(dir+"\n"), 0o644)
}

// candidateDshHomeRoots returns install/data homes to probe for Desktop Host entries.
// Order: explicit DSH_HOME, persisted user-chosen dir, then common under $HOME / XDG.
func candidateDshHomeRoots() []string {
	var roots []string
	if h := strings.TrimSpace(os.Getenv("DSH_HOME")); h != "" {
		roots = appendUnique(roots, h)
	}
	if chosen := loadUserChosenDshHome(); chosen != "" {
		roots = appendUnique(roots, chosen)
	}
	home := userHomeDir()
	if home == "" {
		return roots
	}
	xdgData := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if xdgData == "" {
		xdgData = filepath.Join(home, ".local", "share")
	}
	roots = appendUnique(roots,
		filepath.Join(home, ".dsh"),
		filepath.Join(home, "dsh"),
		filepath.Join(home, ".local", "share", "dsh"),
		filepath.Join(xdgData, "dsh"),
		filepath.Join(home, ".local", "opt", "dsh"),
	)
	return roots
}

// relativeHostEntries under a dsh home / install prefix (user install only).
func relativeHostEntries() []string {
	return []string{
		filepath.Join("bin", "dsh-desktop"),
		filepath.Join("bin", "dsh-plugin-desktop"),
		filepath.Join("bin", "dsh"),
		filepath.Join("node_modules", "@deepseek-ai", "dsh", "lib", "bin.js"),
		filepath.Join("lib", "host-main.js"),
		filepath.Join("dsh-plugin-desktop", "lib", "host-main.js"),
		filepath.Join("lib", "bin.js"),
	}
}

// probeHomeRoot looks for a Desktop Host entry under root.
func probeHomeRoot(root, label string, checked *[]string) (hit *hostDiscoverHit, ok bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, false
	}
	for _, rel := range relativeHostEntries() {
		p := filepath.Join(root, rel)
		*checked = appendUnique(*checked, p)
		if !fileExists(p) {
			continue
		}
		cmd, err := commandFromHostBin(p)
		if err != nil {
			continue
		}
		return &hostDiscoverHit{
			Command: cmd,
			Reason:  label + ":" + rel,
			Path:    p,
		}, true
	}
	return nil, false
}

// probeLocalBin checks ~/.local/bin shims directly (common user install).
func probeLocalBin(checked *[]string) (hit *hostDiscoverHit, ok bool) {
	home := userHomeDir()
	if home == "" {
		return nil, false
	}
	dir := filepath.Join(home, ".local", "bin")
	for _, name := range []string{"dsh-desktop", "dsh-plugin-desktop"} {
		p := filepath.Join(dir, name)
		*checked = appendUnique(*checked, p)
		if !fileExists(p) {
			continue
		}
		cmd, err := commandFromHostBin(p)
		if err != nil {
			continue
		}
		return &hostDiscoverHit{
			Command: cmd,
			Reason:  "HOME:~/.local/bin/" + name,
			Path:    p,
		}, true
	}
	return nil, false
}

func probePATH(checked *[]string) (hit *hostDiscoverHit, ok bool) {
	for _, name := range []string{"dsh-desktop", "dsh-plugin-desktop", "dsh"} {
		*checked = appendUnique(*checked, "PATH:"+name)
		p, err := exec.LookPath(name)
		if err != nil || p == "" {
			continue
		}
		*checked = appendUnique(*checked, p)
		cmd, err := commandFromHostBin(p)
		if err != nil {
			continue
		}
		return &hostDiscoverHit{
			Command: cmd,
			Reason:  "PATH:" + name,
			Path:    p,
		}, true
	}
	return nil, false
}

func friendlyHostMissingMessage(checked []string) string {
	var b strings.Builder
	b.WriteString("未找到可用的 Desktop Host（Cordis Host）。\n")
	b.WriteString("No usable Desktop Host was found for the Wails shell.\n\n")
	b.WriteString("已检查 / Checked paths:\n")
	if len(checked) == 0 {
		b.WriteString("  (none)\n")
	} else {
		limit := len(checked)
		if limit > 24 {
			limit = 24
		}
		for _, p := range checked[:limit] {
			b.WriteString("  - ")
			b.WriteString(p)
			b.WriteByte('\n')
		}
		if len(checked) > limit {
			fmt.Fprintf(&b, "  … and %d more\n", len(checked)-limit)
		}
	}
	b.WriteString("\n下一步 / Next steps:\n")
	b.WriteString("  1. 安装用户级 dsh（@deepseek-ai/dsh）到家目录，例如 ~/.dsh 或 ~/dsh；或在应用内「选择目录…」\n")
	b.WriteString("     Install user-level dsh (@deepseek-ai/dsh) under home (e.g. ~/.dsh or ~/dsh); or use in-app Choose directory…\n")
	b.WriteString("  2. 确保存在 bin/dsh、bin/dsh-desktop，或 node_modules/@deepseek-ai/dsh/lib/bin.js\n")
	b.WriteString("     Ensure bin/dsh, bin/dsh-desktop, or node_modules/@deepseek-ai/dsh/lib/bin.js exists\n")
	b.WriteString("  3. 或设置环境变量 / Or set:\n")
	b.WriteString("       export DSH_HOME=$HOME/.dsh\n")
	b.WriteString("       export DSH_BIN=/path/to/dsh   # or host-main.js / dsh-desktop\n")
	b.WriteString("  4. 把 dsh 放到 PATH / Or put dsh on PATH (~/.local/bin)\n")
	b.WriteString("  5. 已有运行中的 Host 时可设置 DSH_HOST_URL / Or attach with DSH_HOST_URL\n")
	b.WriteString("\n说明: 桌面 Wails 只附着用户安装的 dsh Host（同 `dsh --profile <your|web>`）；默认不使用包内/monorepo host-main。\n")
	b.WriteString("Note: Wails only attaches a user-installed dsh Host (same as `dsh --profile <your|web>`); packaged/monorepo host-main is off by default (DSH_ALLOW_PACKAGED_HOST=1 for dev only).\n")
	return b.String()
}

// ProbeHostDiscovery probes explicit DSH_BIN and user-home/PATH Host installs (not monorepo).
// Order: DSH_BIN → DSH_HOME → persisted user-chosen dir → home installs → ~/.local/bin → PATH.
// Packaged/monorepo Host is never part of this probe (opt-in only via DSH_ALLOW_PACKAGED_HOST).
func ProbeHostDiscovery() hostDiscoverReport {
	var checked []string
	report := hostDiscoverReport{}

	if bin := strings.TrimSpace(os.Getenv("DSH_BIN")); bin != "" {
		checked = appendUnique(checked, "DSH_BIN="+bin)
		if cmd, err := commandFromHostBin(bin); err == nil {
			report.Hit = &hostDiscoverHit{Command: cmd, Reason: "DSH_BIN", Path: bin}
			report.Checked = checked
			report.Message = "Desktop Host via DSH_BIN: " + bin
			return report
		}
	} else {
		checked = appendUnique(checked, "DSH_BIN=(unset)")
	}

	dshHome := strings.TrimSpace(os.Getenv("DSH_HOME"))
	dshHomeUnusable := false
	if dshHome != "" {
		if hit, ok := probeHomeRoot(dshHome, "DSH_HOME", &checked); ok {
			report.Hit = hit
			report.Checked = checked
			report.Message = "Desktop Host via " + hit.Reason + " → " + hit.Path
			return report
		}
		dshHomeUnusable = true
		checked = appendUnique(checked, "DSH_HOME=(set but no usable Host entry):"+dshHome)
	} else {
		checked = appendUnique(checked, "DSH_HOME=(unset)")
	}

	for _, root := range candidateDshHomeRoots() {
		if dshHome != "" && filepath.Clean(root) == filepath.Clean(dshHome) {
			continue
		}
		label := "HOME"
		home := userHomeDir()
		if home != "" {
			if rel, err := filepath.Rel(home, root); err == nil && !strings.HasPrefix(rel, "..") {
				label = "HOME:~/" + filepath.ToSlash(rel)
			} else {
				label = "HOME:" + root
			}
		}
		if hit, ok := probeHomeRoot(root, label, &checked); ok {
			report.Hit = hit
			report.Checked = checked
			report.Message = "Desktop Host via " + hit.Reason + " → " + hit.Path
			return report
		}
	}

	if hit, ok := probeLocalBin(&checked); ok {
		report.Hit = hit
		report.Checked = checked
		report.Message = "Desktop Host via " + hit.Reason + " → " + hit.Path
		return report
	}

	if hit, ok := probePATH(&checked); ok {
		report.Hit = hit
		report.Checked = checked
		report.Message = "Desktop Host via " + hit.Reason + " → " + hit.Path
		return report
	}

	report.Checked = checked
	if dshHomeUnusable {
		report.Message = friendlyInvalidHomeMessage(dshHome, "DSH_HOME") + "\n" + friendlyHostMissingMessage(checked)
	} else {
		report.Message = friendlyHostMissingMessage(checked)
	}
	return report
}

// discoverUserInstalledHostCommand finds a user-installed Desktop Host
// (user install: DSH_BIN / DSH_HOME / ~/.dsh|~/dsh|XDG / ~/.local/bin / PATH).
func discoverUserInstalledHostCommand() (command string, reason string, ok bool) {
	rep := ProbeHostDiscovery()
	if rep.Hit == nil {
		return "", "", false
	}
	return rep.Hit.Command, rep.Hit.Reason, true
}

func defaultHostBootstrap() (command string, urlFile string, err error) {
	urlFile = defaultHostURLFile()

	// Product default: user-installed Host only (DSH_BIN is covered inside ProbeHostDiscovery).
	if cmd, reason, ok := discoverUserInstalledHostCommand(); ok {
		_ = reason
		return cmd, urlFile, nil
	}

	missing := ProbeHostDiscovery()

	// Packaged / monorepo Host is DEV-ONLY (default OFF).
	if !envTruthy("DSH_ALLOW_PACKAGED_HOST") {
		return "", "", fmt.Errorf("%s", missing.Message)
	}

	_, pluginDir, repoDir, layoutErr := locateWailsLayout()
	if layoutErr != nil {
		return "", "", fmt.Errorf("%s\n(packaged layout missing: %v)", missing.Message, layoutErr)
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
	return "", "", fmt.Errorf("%s\n(packaged Host incomplete: need %s or %s)", missing.Message, yarnLock, binJS)
}
