package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ShellService exposes native shell controls to the embedded control UI and to
// future Cordis Host bridge callers. Methods are discovered by
// `wails3 generate bindings`.
type ShellService struct {
	mu               sync.Mutex
	app              *application.App
	window           *application.WebviewWindow
	hostURL          string
	sidecar          *HostSidecar
	aux              *AuxWindowService
	bridge           *BridgeService
	lastHostError    string
	lastHostKind     string
	relaunchAttempts int
	relaunching      bool
	suppressRelaunch bool
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
	s.ExpectHostExit()
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

// ExpectHostExit suppresses auto-relaunch for the next intentional Host stop/quit.
func (s *ShellService) ExpectHostExit() {
	s.mu.Lock()
	s.suppressRelaunch = true
	s.relaunching = false
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar != nil {
		sidecar.ExpectExit()
	}
}

// ClearHostExitExpectation re-enables auto-relaunch (e.g. after manual Retry).
func (s *ShellService) ClearHostExitExpectation() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suppressRelaunch = false
}

// RelaunchStatus returns a short diagnostic for tests / control UI.
func (s *ShellService) RelaunchStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("relaunching=%v attempts=%d suppress=%v", s.relaunching, s.relaunchAttempts, s.suppressRelaunch)
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
// Empty cancel returns ("", nil). Non-directory / empty path errors. A directory without a
// Host entry is still persisted (install-into-folder), but StartHostSidecar will classify
// it as not-usable / invalid-home and show the friendly host-error page.
func (s *ShellService) ChooseUserDshHomeDirectory() (string, error) {
	path, err := s.OpenDirectoryDialog()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	st, statErr := os.Stat(path)
	if statErr != nil || !st.IsDir() {
		msg := friendlyInvalidHomeMessage(path, "chosen")
		s.rememberHostFailure(hostFailInvalidHome, msg)
		_ = s.ShowHostErrorPage(hostFailInvalidHome, msg)
		return "", fmt.Errorf("%s", msg)
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

// LastHostErrorDetail returns the most recent Host failure detail for host-error.html.
func (s *ShellService) LastHostErrorDetail() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastHostError != "" {
		return s.lastHostError
	}
	if s.sidecar != nil {
		return s.sidecar.LastError()
	}
	return ""
}

// LastHostErrorKind returns the classified failure kind for the last Host error.
func (s *ShellService) LastHostErrorKind() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastHostKind
}

func (s *ShellService) rememberHostFailure(kind, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastHostKind = kind
	s.lastHostError = detail
}

// ShowHostErrorPage loads the friendly abnormal-scenario page into the main window.
func (s *ShellService) ShowHostErrorPage(kind, detail string) error {
	if strings.TrimSpace(kind) == "" {
		kind = hostFailGeneric
	}
	s.rememberHostFailure(kind, detail)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return fmt.Errorf("main window is not attached")
	}
	s.hostURL = ""
	s.window.SetURL(hostErrorPageURL(kind, detail))
	s.window.SetTitle("DSH Desktop — Host problem")
	s.window.Show()
	s.window.Focus()
	return nil
}

// ShowHostFailurePage classifies errText and shows install-help or host-error.
func (s *ShellService) ShowHostFailurePage(errText string) error {
	errText = strings.TrimSpace(errText)
	var err error
	if errText != "" {
		err = fmt.Errorf("%s", errText)
	}
	rep := ProbeHostDiscovery()
	kind := classifyHostFailure(err, &rep)
	if kind == hostFailMissing {
		s.rememberHostFailure(kind, errText)
		return s.ShowHostInstallPage()
	}
	if kind == "" {
		kind = hostFailGeneric
	}
	detail := errText
	if detail == "" {
		detail = rep.Message
	}
	return s.ShowHostErrorPage(kind, detail)
}

// StartHostSidecar starts/discovers the Cordis Host and navigates to its UI URL.
// Pass an empty url to use DSH_HOST_URL / DSH_HOST_COMMAND / DSH_HOST_URL_FILE.
func (s *ShellService) StartHostSidecar(url string) (string, error) {
	s.mu.Lock()
	sidecar := s.sidecar
	// Manual / scheduled start should accept unexpected exits again.
	if !s.relaunching {
		s.suppressRelaunch = false
	}
	s.mu.Unlock()
	if sidecar == nil {
		return "", fmt.Errorf("host sidecar is not configured")
	}
	ready, err := sidecar.Start(url)
	if err != nil {
		rep := ProbeHostDiscovery()
		kind := classifyHostFailure(err, &rep)
		s.rememberHostFailure(kind, err.Error())
		return "", err
	}
	// Fresh READY — if we are not in a crash storm loop, allow counter reset on stable uptime
	// (handled in HandleUnexpectedHostExit via ReadyAt). Manual retry from host-error clears storm.
	s.mu.Lock()
	if !s.relaunching {
		s.relaunchAttempts = 0
	}
	s.mu.Unlock()
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
// Intentional stop does not auto-relaunch.
func (s *ShellService) StopHostSidecar() error {
	s.ExpectHostExit()
	s.mu.Lock()
	sidecar := s.sidecar
	s.mu.Unlock()
	if sidecar == nil {
		return nil
	}
	return sidecar.Stop()
}

// ShowHostRestartingPage shows a brief in-window status while auto-relaunch runs.
func (s *ShellService) ShowHostRestartingPage(attempt, max int, detail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == nil {
		return fmt.Errorf("main window is not attached")
	}
	s.hostURL = ""
	s.window.SetURL(hostRestartingPageURL(attempt, max, detail))
	s.window.SetTitle(hostRelaunchStatusLine(attempt, max))
	s.window.Show()
	s.window.Focus()
	return nil
}

// HandleUnexpectedHostExit runs backoff auto-relaunch, then host-error on exhaustion.
// Safe to call from the HostSidecar OnUnexpectedExit callback (non-blocking).
func (s *ShellService) HandleUnexpectedHostExit(err error) {
	if err == nil {
		return
	}
	policy := loadHostRelaunchPolicy()
	s.mu.Lock()
	if s.suppressRelaunch {
		s.mu.Unlock()
		log.Printf("dsh-wails-shell: host exit ignored (intentional stop/quit): %v", err)
		return
	}
	if s.relaunching {
		s.mu.Unlock()
		log.Printf("dsh-wails-shell: host exit while relaunch in progress: %v", err)
		return
	}
	sidecar := s.sidecar
	attempts := s.relaunchAttempts
	s.mu.Unlock()

	uptime := time.Duration(0)
	if sidecar != nil {
		if readyAt := sidecar.ReadyAt(); !readyAt.IsZero() {
			uptime = time.Since(readyAt)
		}
	}
	should, attempt, delay := policy.nextRelaunchAttempt(attempts, uptime)
	if !should {
		log.Printf("dsh-wails-shell: host relaunch exhausted (attempts=%d uptime=%s): %v", attempts, uptime, err)
		s.mu.Lock()
		s.relaunchAttempts = attempts
		s.relaunching = false
		s.mu.Unlock()
		if pageErr := s.ShowHostFailurePage(err.Error()); pageErr != nil {
			log.Printf("dsh-wails-shell: host-error page: %v", pageErr)
		}
		return
	}

	s.mu.Lock()
	s.relaunching = true
	s.relaunchAttempts = attempt
	s.mu.Unlock()
	log.Printf("dsh-wails-shell: host unexpected exit; auto-relaunch %d/%d in %s: %v", attempt, policy.MaxAttempts, delay, err)
	_ = s.ShowHostRestartingPage(attempt, policy.MaxAttempts, err.Error())

	go s.runHostRelaunch(attempt, policy.MaxAttempts, delay, err.Error())
}

func (s *ShellService) runHostRelaunch(attempt, max int, delay time.Duration, lastErr string) {
	if delay > 0 {
		time.Sleep(delay)
	}
	s.mu.Lock()
	suppress := s.suppressRelaunch
	s.mu.Unlock()
	if suppress {
		log.Printf("dsh-wails-shell: aborting auto-relaunch (intentional stop)")
		s.mu.Lock()
		s.relaunching = false
		s.mu.Unlock()
		return
	}
	// Clear suppress from a prior Stop so Start can own a fresh generation.
	s.ClearHostExitExpectation()
	ready, startErr := s.StartHostSidecar("")
	s.mu.Lock()
	s.relaunching = false
	if startErr == nil {
		// Successful relaunch — keep attempt count until stable uptime resets on next crash.
		s.suppressRelaunch = false
		s.mu.Unlock()
		log.Printf("dsh-wails-shell: host auto-relaunch succeeded (%d/%d) → %s", attempt, max, ready)
		return
	}
	attempts := s.relaunchAttempts
	s.mu.Unlock()
	log.Printf("dsh-wails-shell: host auto-relaunch failed (%d/%d): %v", attempt, max, startErr)
	policy := loadHostRelaunchPolicy()
	should, next, nextDelay := policy.nextRelaunchAttempt(attempts, 0)
	if should {
		s.mu.Lock()
		s.relaunching = true
		s.relaunchAttempts = next
		s.mu.Unlock()
		_ = s.ShowHostRestartingPage(next, policy.MaxAttempts, startErr.Error())
		s.runHostRelaunch(next, policy.MaxAttempts, nextDelay, startErr.Error())
		return
	}
	detail := startErr.Error()
	if lastErr != "" {
		detail = lastErr + "\n\nRelaunch failed: " + detail
	}
	if pageErr := s.ShowHostFailurePage(detail); pageErr != nil {
		log.Printf("dsh-wails-shell: host-error page: %v", pageErr)
	}
}

// ResetHostRelaunchAttempts clears the crash-storm counter (manual Retry / stable recovery).
func (s *ShellService) ResetHostRelaunchAttempts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relaunchAttempts = 0
	s.relaunching = false
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
