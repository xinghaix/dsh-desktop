package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ShellService exposes native shell controls to the embedded control UI and to
// future Cordis Host bridge callers. Methods are discovered by
// `wails3 generate bindings`.
type ShellService struct {
	mu      sync.Mutex
	app     *application.App
	window  *application.WebviewWindow
	hostURL string
	sidecar *HostSidecar
	aux     *AuxWindowService
	bridge  *BridgeService
}

func NewShellService(sidecar *HostSidecar) *ShellService {
	if sidecar == nil {
		sidecar = NewHostSidecar()
	}
	return &ShellService{sidecar: sidecar}
}

func (s *ShellService) attach(app *application.App, window *application.WebviewWindow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.app = app
	s.window = window
}

// setInitialHostURL records the URL passed on the CLI before the window finishes loading.
func (s *ShellService) setInitialHostURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostURL = url
}

// Status returns a short diagnostic string for the control panel.
func (s *ShellService) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	url := s.hostURL
	if url == "" {
		url = "(embedded control UI)"
	}
	return fmt.Sprintf("DSH Wails shell ready; url=%s", url)
}

// CurrentURL returns the URL currently scheduled for the main window.
func (s *ShellService) CurrentURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hostURL
}

// LoadHostURL navigates the main window to a Cordis Host / desktop UI origin
// (typically http://127.0.0.1:<port>/ from desktopLoopbackBrowserUrl).
func (s *ShellService) LoadHostURL(url string) error {
	if url == "" {
		return fmt.Errorf("url must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return fmt.Errorf("main window is not attached")
	}
	s.hostURL = url
	s.window.SetURL(url)
	s.window.SetTitle("DSH Desktop")
	s.window.Show()
	s.window.Focus()
	return nil
}

// ShowControlUI reloads the embedded Wails control page (asset server "/").
func (s *ShellService) ShowControlUI() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return fmt.Errorf("main window is not attached")
	}
	s.hostURL = ""
	s.window.SetURL("/")
	s.window.SetTitle("DSH Desktop — Wails shell")
	s.window.Show()
	s.window.Focus()
	return nil
}

// FlashMainWindow requests dock/taskbar attention (macOS Dock bounce / Windows flash).
func (s *ShellService) FlashMainWindow(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return
	}
	s.window.Flash(enabled)
}

// ShowWindow focuses the main window.
func (s *ShellService) ShowWindow() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return
	}
	s.window.Show()
	s.window.Focus()
}

// HideWindow hides the main window (tray keeps the process alive).
func (s *ShellService) HideWindow() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return
	}
	s.window.Hide()
}

// OpenDirectoryDialog opens a native directory picker and returns the path.
// Cancel returns ("", nil) on Linux/macOS, or an error whose message contains
// "cancel" on Windows. Callers should treat both as user cancel, not failure.
func (s *ShellService) OpenDirectoryDialog() (string, error) {
	s.mu.Lock()
	app := s.app
	window := s.window
	s.mu.Unlock()
	if app == nil {
		return "", fmt.Errorf("application is not attached")
	}
	dialog := app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("Select directory")
	if window != nil {
		dialog = dialog.AttachToWindow(window)
	}
	return dialog.PromptForSingleSelection()
}

// OpenFileDialog opens a native file picker and returns the selected path.
func (s *ShellService) OpenFileDialog() (string, error) {
	s.mu.Lock()
	app := s.app
	window := s.window
	s.mu.Unlock()
	if app == nil {
		return "", fmt.Errorf("application is not attached")
	}
	dialog := app.Dialog.OpenFile().
		CanChooseDirectories(false).
		CanChooseFiles(true).
		SetTitle("Select file")
	if window != nil {
		dialog = dialog.AttachToWindow(window)
	}
	return dialog.PromptForSingleSelection()
}

// ShowInfoDialog shows a native information dialog.
func (s *ShellService) ShowInfoDialog(title, message string) {
	s.mu.Lock()
	app := s.app
	window := s.window
	s.mu.Unlock()
	if app == nil {
		return
	}
	d := app.Dialog.Info().SetTitle(title).SetMessage(message)
	if window != nil {
		d = d.AttachToWindow(window)
	}
	d.Show()
}

// Quit requests application shutdown.
func (s *ShellService) Quit() {
	s.mu.Lock()
	app := s.app
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar != nil {
		_ = sidecar.Stop()
	}
	if app != nil {
		app.Quit()
	} else {
		os.Exit(0)
	}
}

// HostSidecarStatus reports Cordis Host sidecar process state.
func (s *ShellService) HostSidecarStatus() string {
	s.mu.Lock()
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar == nil {
		return "sidecar=(nil)"
	}
	return sidecar.Status()
}

// HostDiscoverStatus reports user-home Desktop Host probe for the control UI.
// Includes hit reason/path or a friendly missing message with checked paths.
func (s *ShellService) HostDiscoverStatus() string {
	rep := ProbeHostDiscovery()
	if rep.Hit != nil {
		return fmt.Sprintf("host-discover=ok reason=%s path=%s", rep.Hit.Reason, rep.Hit.Path)
	}
	return rep.Message
}

// ShowHostDiscoverHelp opens the Host install-help page (or status when Host is found).
func (s *ShellService) ShowHostDiscoverHelp() error {
	rep := ProbeHostDiscovery()
	if rep.Hit != nil {
		s.mu.Lock()
		aux := s.aux
		s.mu.Unlock()
		title := "Desktop Host found"
		if aux != nil {
			return aux.OpenInfoDialog(title, rep.Message)
		}
		s.ShowInfoDialog(title, rep.Message)
		return nil
	}
	return s.ShowHostInstallPage()
}

// ShowHostInstallPage loads the polished install-help page into the main window.
func (s *ShellService) ShowHostInstallPage() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return fmt.Errorf("main window is not attached")
	}
	s.hostURL = ""
	s.window.SetURL("/shell-ui/host-install.html")
	s.window.SetTitle("DSH Desktop — Install Host")
	s.window.Show()
	s.window.Focus()
	return nil
}

// ChooseUserDshHomeDirectory opens a folder picker, persists DSH_HOME, and returns the path.
func (s *ShellService) ChooseUserDshHomeDirectory() (string, error) {
	path, err := s.OpenDirectoryDialog()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := saveUserChosenDshHome(path); err != nil {
		return "", err
	}
	_ = os.Setenv("DSH_HOME", path)
	return path, nil
}

// HostDiscoverCheckedPathsJSON returns checked discovery paths for the install page.
func (s *ShellService) HostDiscoverCheckedPaths() string {
	rep := ProbeHostDiscovery()
	if len(rep.Checked) == 0 {
		return "(none)"
	}
	return strings.Join(rep.Checked, "\n")
}

// StartHostSidecar starts/discovers the Cordis Host and navigates to its UI URL.
// Pass an empty url to use DSH_HOST_URL / DSH_HOST_COMMAND / DSH_HOST_URL_FILE.
func (s *ShellService) StartHostSidecar(url string) (string, error) {
	s.mu.Lock()
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar == nil {
		return "", fmt.Errorf("host sidecar is not configured")
	}
	ready, err := sidecar.Start(url)
	if err != nil {
		return "", err
	}
	if ready == recoveryURLSentinel || strings.HasPrefix(ready, "recovery://") {
		// Host is in Recovery RPC keep-alive; AuxWindowService already opened Recovery.
		return ready, nil
	}
	navigate := ready
	s.mu.Lock()
	bridge := s.bridge
	s.mu.Unlock()
	if bridge != nil {
		if proxied, perr := bridge.PreferProxiedHostURL(ready); perr == nil && proxied != "" {
			navigate = proxied
		} else if perr != nil {
			fmt.Printf("dsh-wails-shell: auth proxy unavailable, loading Host URL directly: %v\n", perr)
		}
	}
	if err := s.LoadHostURL(navigate); err != nil {
		return navigate, err
	}
	return navigate, nil
}

// StopHostSidecar stops a spawned Cordis Host process, if any.
func (s *ShellService) StopHostSidecar() error {
	s.mu.Lock()
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar == nil {
		return nil
	}
	return sidecar.Stop()
}

// RecoveryRPC returns the Host Recovery RPC client when the sidecar announced one.
func (s *ShellService) RecoveryRPC() *RecoveryRpcClient {
	s.mu.Lock()
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar == nil {
		return nil
	}
	return sidecar.RecoveryRPC()
}

// Preferred profile / safe-mode hints for Host sidecar relaunches (hybrid).
func (s *ShellService) setPreferredProfile(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sidecar != nil {
		s.sidecar.SetPreferredProfile(name)
	}
}

func (s *ShellService) setSafeMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sidecar != nil {
		s.sidecar.SetSafeMode(enabled)
	}
}

func (s *ShellService) attachAux(aux *AuxWindowService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.aux = aux
}

func (s *ShellService) attachBridge(bridge *BridgeService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bridge = bridge
}

// OpenSetupWizard opens the Wails setup wizard auxiliary window.
func (s *ShellService) OpenSetupWizard() error {
	s.mu.Lock()
	aux := s.aux
	s.mu.Unlock()
	if aux == nil {
		return fmt.Errorf("aux windows not attached")
	}
	return aux.OpenSetupWizard()
}

// OpenProfileSelector opens the Wails profile selection auxiliary window.
func (s *ShellService) OpenProfileSelector() error {
	s.mu.Lock()
	aux := s.aux
	s.mu.Unlock()
	if aux == nil {
		return fmt.Errorf("aux windows not attached")
	}
	return aux.OpenProfileSelector()
}

// OpenRecovery opens the Wails startup recovery auxiliary window.
func (s *ShellService) OpenRecovery(detail string) error {
	s.mu.Lock()
	aux := s.aux
	s.mu.Unlock()
	if aux == nil {
		return fmt.Errorf("aux windows not attached")
	}
	return aux.OpenRecovery(detail)
}
