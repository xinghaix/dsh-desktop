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

// HostSidecar optionally starts the Cordis Host (Node / Electron RunAsNode) and
// discovers the loopback UI URL that Electron historically passed to loadURL.
//
// Discovery order for Start:
//  1. Explicit URL argument / DSH_HOST_URL
//  2. URL file (DSH_HOST_URL_FILE) written by the host when the web server is up
//  3. Stdout line "DSH_HOST_READY http://127.0.0.1:PORT/" from DSH_HOST_COMMAND
type HostSidecar struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	url     string
	running bool
	lastErr string
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
	return fmt.Sprintf("sidecar=%s url=%s err=%s", state, url, err)
}

// CurrentURL returns the discovered Host UI URL, if any.
func (h *HostSidecar) CurrentURL() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
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
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Env = append(os.Environ(),
		"DSH_WAILS_HOST_SIDECAR=1",
	)
	if urlFile != "" {
		cmd.Env = append(cmd.Env, "DSH_HOST_URL_FILE="+urlFile)
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
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, "[dsh-host]", line)
			if strings.HasPrefix(line, hostReadyPrefix) {
				u := strings.TrimSpace(strings.TrimPrefix(line, hostReadyPrefix))
				if u != "" {
					h.setURL(u)
				}
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
