package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const hostReadyPrefix = "DSH_HOST_READY "
const hostRecoveryRequiredPrefix = "DSH_HOST_RECOVERY_REQUIRED "

// HostSidecar optionally starts the Cordis Host (Node / Electron RunAsNode) and
// discovers the loopback UI URL that Electron historically passed to loadURL.
//
// Discovery order for Start:
//  1. Explicit URL argument / DSH_HOST_URL
//  2. URL file (DSH_HOST_URL_FILE) written by the host when the web server is up
//  3. Stdout line "DSH_HOST_READY http://127.0.0.1:PORT/" from DSH_HOST_COMMAND
type HostSidecar struct {
	mu               sync.Mutex
	cmd              *exec.Cmd
	cancel           context.CancelFunc
	url              string
	running          bool
	lastErr          string
	preferredProfile string
	safeMode         bool
	bridge           *BridgeService
	aux              *AuxWindowService
	caps             *CapabilitiesService
	recoveryDetail   string
	recoveryRPC      *RecoveryRpcClient
}

func NewHostSidecar() *HostSidecar {
	return &HostSidecar{}
}

// Status reports whether a sidecar process is running and which URL was found.
func (h *HostSidecar) Status() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	state := "stopped"
	if h.running {
		state = "running"
	}
	url := h.url
	if url == "" {
		url = "(none)"
	}
	err := h.lastErr
	if err == "" {
		err = "(none)"
	}
	recovery := h.recoveryDetail
	if recovery == "" {
		recovery = "(none)"
	}
	rpc := "(none)"
	if h.recoveryRPC != nil && h.recoveryRPC.BaseURL != "" {
		rpc = h.recoveryRPC.BaseURL
	}
	return fmt.Sprintf("sidecar=%s url=%s err=%s recovery=%s rpc=%s", state, url, err, recovery, rpc)
}

// CurrentURL returns the discovered Host UI URL, if any.
func (h *HostSidecar) CurrentURL() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
}

// RecoveryRPC returns the Host Recovery RPC client when announced.
func (h *HostSidecar) RecoveryRPC() *RecoveryRpcClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.recoveryRPC
}

// SetPreferredProfile asks the next Host spawn to prefer this profile name.
func (h *HostSidecar) SetPreferredProfile(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.preferredProfile = strings.TrimSpace(name)
}

// SetSafeMode asks the next Host spawn to enter disposable Safe Mode.
func (h *HostSidecar) SetSafeMode(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.safeMode = enabled
}

// AttachBridge lets Host stdout auth-header lines update the BridgeService.
func (h *HostSidecar) AttachBridge(bridge *BridgeService) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.bridge = bridge
}

// AttachAux lets Host recovery announcements open the Wails Recovery Assistant.
func (h *HostSidecar) AttachAux(aux *AuxWindowService) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.aux = aux
}

// AttachCaps lets Host LAN HTTPS announce lines update CapabilitiesService.
func (h *HostSidecar) AttachCaps(caps *CapabilitiesService) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.caps = caps
}

// Start discovers or launches the Cordis Host and returns the UI URL.
func (h *HostSidecar) Start(explicitURL string) (string, error) {
	if u := strings.TrimSpace(explicitURL); u != "" {
		h.setURL(u)
		return u, nil
	}
	if u := strings.TrimSpace(os.Getenv("DSH_HOST_URL")); u != "" {
		h.setURL(u)
		return u, nil
	}

	urlFile := strings.TrimSpace(os.Getenv("DSH_HOST_URL_FILE"))
	command := strings.TrimSpace(os.Getenv("DSH_HOST_COMMAND"))
	if command == "" && urlFile == "" {
		if !hostAutostartEnabled() {
			return "", fmt.Errorf("host autostart disabled (DSH_HOST_AUTOSTART=0); set DSH_HOST_URL or DSH_HOST_COMMAND")
		}
		boot, file, err := defaultHostBootstrap()
		if err != nil {
			return "", err
		}
		command = boot
		urlFile = file
	}
	if urlFile == "" {
		urlFile = defaultHostURLFile()
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		cancel()
		if h.url != "" {
			return h.url, nil
		}
		return "", fmt.Errorf("sidecar already starting")
	}
	h.cancel = cancel
	h.running = true
	h.lastErr = ""
	h.mu.Unlock()

	if command != "" {
		if err := h.spawn(ctx, command, urlFile); err != nil {
			h.fail(err)
			return "", err
		}
	}

	deadline := 180 * time.Second
	if v := strings.TrimSpace(os.Getenv("DSH_HOST_READY_TIMEOUT")); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			deadline = parsed
		}
	}

	url, err := h.waitForURL(ctx, urlFile, deadline)
	if err != nil {
		h.fail(err)
		_ = h.Stop()
		return "", err
	}
	h.setURL(url)
	return url, nil
}

// Stop terminates a spawned sidecar process, if any.
func (h *HostSidecar) Stop() error {
	h.mu.Lock()
	cancel := h.cancel
	cmd := h.cmd
	h.cancel = nil
	h.cmd = nil
	h.running = false
	h.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return cmd.Process.Kill()
	}
}

func (h *HostSidecar) spawn(ctx context.Context, command, urlFile string) error {
	h.mu.Lock()
	profile := h.preferredProfile
	safe := h.safeMode
	bridge := h.bridge
	h.safeMode = false // one-shot
	h.mu.Unlock()
	if safe {
		// Electron Host reads the exact argv marker --dsh-desktop-safe-mode.
		command = command + " --dsh-desktop-safe-mode"
	}
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Env = append(os.Environ(), "DSH_WAILS_HOST_SIDECAR=1")
	if urlFile != "" {
		cmd.Env = append(cmd.Env, "DSH_HOST_URL_FILE="+urlFile)
	}
	if profile != "" {
		cmd.Env = append(cmd.Env, "DSH_DESKTOP_DEFAULT_PROFILE="+profile)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	h.mu.Lock()
	h.cmd = cmd
	h.mu.Unlock()

	go func() {
		scanner := bufio.NewScanner(stdout)
		// Auth header lines can be long; raise the token limit slightly.
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, "[dsh-host]", line)
			if strings.HasPrefix(line, hostReadyPrefix) {
				u := strings.TrimSpace(strings.TrimPrefix(line, hostReadyPrefix))
				if u != "" {
					h.setURL(u)
				}
			}
			if strings.HasPrefix(line, hostRecoveryRequiredPrefix) {
				detail := strings.TrimSpace(strings.TrimPrefix(line, hostRecoveryRequiredPrefix))
				h.mu.Lock()
				h.recoveryDetail = detail
				aux := h.aux
				h.mu.Unlock()
				if aux != nil {
					_ = aux.OpenRecovery(detail)
				}
			}
			if baseURL, token, ok := ParseRecoveryRpcAnnounceLine(line); ok {
				client := &RecoveryRpcClient{BaseURL: baseURL, Token: token}
				h.mu.Lock()
				h.recoveryRPC = client
				aux := h.aux
				h.mu.Unlock()
				if aux != nil {
					aux.AttachRecoveryRPC(client)
				}
			}
			if bridge != nil {
				_ = bridge.IngestHostAuthHeaderLine(line)
			}
			h.mu.Lock()
			caps := h.caps
			h.mu.Unlock()
			if caps != nil {
				_ = caps.IngestLanHttpsAnnounceLine(line)
			}
		}
	}()
	go func() {
		err := cmd.Wait()
		h.mu.Lock()
		h.running = false
		if err != nil && h.lastErr == "" {
			h.lastErr = err.Error()
		}
		h.mu.Unlock()
	}()
	return nil
}

func (h *HostSidecar) waitForURL(ctx context.Context, urlFile string, deadline time.Duration) (string, error) {
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		if u := h.CurrentURL(); u != "" {
			return u, nil
		}
		h.mu.Lock()
		hasRPC := h.recoveryRPC != nil && h.recoveryRPC.BaseURL != ""
		h.mu.Unlock()
		if hasRPC {
			// Recovery keep-alive: Host may never announce DSH_HOST_READY.
			return recoveryURLSentinel, nil
		}
		if urlFile != "" {
			if raw, err := os.ReadFile(urlFile); err == nil {
				u := strings.TrimSpace(string(raw))
				if u != "" {
					return u, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("host sidecar cancelled")
		case <-timer.C:
			return "", fmt.Errorf("timed out waiting for Cordis Host UI URL")
		case <-tick.C:
		}
	}
}

func (h *HostSidecar) setURL(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.url = url
}

func (h *HostSidecar) fail(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastErr = err.Error()
	h.running = false
}
