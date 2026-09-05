package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Host failure kinds shown on /shell-ui/host-error.html.
const (
	hostFailMissing     = "missing"
	hostFailInvalidHome = "invalid-home"
	hostFailNotUsable   = "not-usable"
	hostFailTimeout     = "timeout"
	hostFailStart       = "start-failed"
	hostFailCrash       = "crash"
	hostFailPortBind    = "port-bind"
	hostFailProfile     = "profile"
	hostFailGeneric     = "host-error"
)

func classifyHostFailure(err error, discover *hostDiscoverReport) string {
	if err == nil {
		return hostFailGeneric
	}
	msg := err.Error()
	lower := strings.ToLower(msg)

	if strings.Contains(lower, "eaddrinuse") ||
		strings.Contains(lower, "address already in use") ||
		(strings.Contains(lower, "port") && strings.Contains(lower, "bind")) {
		return hostFailPortBind
	}
	if (strings.Contains(lower, "profile") && strings.Contains(lower, "creat") && strings.Contains(lower, "fail")) ||
		strings.Contains(lower, "failed to create web profile") ||
		strings.Contains(lower, "profile fallback") ||
		strings.Contains(lower, "web profile create failed") {
		return hostFailProfile
	}
	if strings.Contains(lower, "host exited") ||
		strings.Contains(lower, "host crashed") ||
		strings.Contains(lower, "signal:") ||
		strings.Contains(lower, "unexpected exit") {
		return hostFailCrash
	}
	if strings.Contains(lower, "failed to start") ||
		strings.Contains(lower, "executable file not found") ||
		(strings.Contains(lower, "no such file") && strings.Contains(lower, "spawn")) {
		return hostFailStart
	}
	if strings.Contains(lower, "timed out waiting for cordis host") ||
		strings.Contains(lower, "ready timeout") {
		return hostFailTimeout
	}

	if discover != nil && discover.Hit == nil {
		dshHome := strings.TrimSpace(os.Getenv("DSH_HOME"))
		chosen := loadUserChosenDshHome()
		if dshHome != "" || chosen != "" {
			root := dshHome
			if root == "" {
				root = chosen
			}
			if st, e := os.Stat(root); e == nil && st.IsDir() {
				return hostFailNotUsable
			}
			return hostFailInvalidHome
		}
		if strings.Contains(msg, "未找到可用的 Desktop Host") ||
			strings.Contains(lower, "no usable desktop host") {
			return hostFailMissing
		}
	}
	if strings.Contains(msg, "未找到可用的 Desktop Host") ||
		strings.Contains(lower, "no usable desktop host") {
		return hostFailMissing
	}
	return hostFailGeneric
}

func friendlyInvalidHomeMessage(root, source string) string {
	var b strings.Builder
	b.WriteString("指定的 dsh 安装目录不可用。\n")
	b.WriteString("The configured dsh install directory is not usable.\n\n")
	fmt.Fprintf(&b, "来源 / Source: %s\n", source)
	fmt.Fprintf(&b, "路径 / Path: %s\n\n", root)
	if strings.TrimSpace(root) == "" {
		b.WriteString("目录为空。请取消空的 DSH_HOME，或选择一个真实安装目录。\n")
		b.WriteString("Path is empty. Unset empty DSH_HOME, or Choose directory…\n")
		return b.String()
	}
	st, err := os.Stat(root)
	if err != nil {
		b.WriteString("路径不存在或无法访问 / Path missing or inaccessible: ")
		b.WriteString(err.Error())
		b.WriteByte('\n')
	} else if !st.IsDir() {
		b.WriteString("路径存在但不是目录 / Path exists but is not a directory.\n")
	} else {
		b.WriteString("目录存在，但未找到可用 Host 入口（bin/dsh、bin/dsh-desktop、node_modules/@deepseek-ai/dsh/lib/bin.js 等）。\n")
		b.WriteString("Directory exists but no usable Host entry was found.\n")
	}
	b.WriteString("\n下一步 / Next steps:\n")
	b.WriteString("  1. 修正 DSH_HOME 或在应用内「选择目录…」\n")
	b.WriteString("  2. 在该目录安装 @deepseek-ai/dsh\n")
	b.WriteString("  3. 或改用 DSH_BIN 指向可执行入口\n")
	return b.String()
}

func hostErrorPageURL(kind, detail string) string {
	q := url.Values{}
	q.Set("kind", kind)
	// Keep query short; full detail is available via ShellService.LastHostErrorDetail.
	if d := strings.TrimSpace(detail); d != "" {
		if len(d) > 400 {
			d = d[:400] + "…"
		}
		q.Set("message", d)
	}
	return "/shell-ui/host-error.html?" + q.Encode()
}

func describeInvalidConfiguredHome() (kind, message string, ok bool) {
	dshHome := strings.TrimSpace(os.Getenv("DSH_HOME"))
	if dshHome != "" {
		if !dirExists(dshHome) {
			return hostFailInvalidHome, friendlyInvalidHomeMessage(dshHome, "DSH_HOME"), true
		}
		var checked []string
		if _, hit := probeHomeRoot(dshHome, "DSH_HOME", &checked); !hit {
			return hostFailNotUsable, friendlyInvalidHomeMessage(dshHome, "DSH_HOME"), true
		}
	}
	chosen := loadUserChosenDshHome()
	if chosen != "" {
		if !dirExists(chosen) {
			return hostFailInvalidHome, friendlyInvalidHomeMessage(chosen, "user-chosen"), true
		}
		var checked []string
		if _, hit := probeHomeRoot(chosen, "user-chosen", &checked); !hit {
			// Only report chosen-dir failure when it was the only configured override
			// and discovery otherwise missed — ProbeHostDiscovery already continues.
			_ = checked
		}
	}
	return "", "", false
}

func validateChosenDshHome(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("%s", friendlyInvalidHomeMessage(abs, "chosen"))
	}
	if !st.IsDir() {
		return fmt.Errorf("%s", friendlyInvalidHomeMessage(abs, "chosen"))
	}
	var checked []string
	if _, ok := probeHomeRoot(abs, "chosen", &checked); !ok {
		return fmt.Errorf("%s", friendlyInvalidHomeMessage(abs, "chosen"))
	}
	return nil
}
