package main

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// AuxWindowService owns setup / profile / recovery webview windows that
// Electron previously opened as BrowserWindow HTML UIs under src/native-ui/.
// These Wails windows are hybrid stand-ins: they talk to Go via bindings and
// cover shell startup flows while Cordis Host still owns full profile CRUD.
type AuxWindowService struct {
	mu      sync.Mutex
	app     *application.App
	shell   *ShellService
	windows map[string]*application.WebviewWindow
	last    AuxWindowResult
}

// AuxWindowResult is the last settled action from an auxiliary window.
type AuxWindowResult struct {
	Kind    string `json:"kind"`
	Action  string `json:"action"`
	Profile string `json:"profile,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

func NewAuxWindowService(shell *ShellService) *AuxWindowService {
	return &AuxWindowService{
		shell:   shell,
		windows: make(map[string]*application.WebviewWindow),
	}
}

func (a *AuxWindowService) attach(app *application.App) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.app = app
}

// LastResult returns the most recent auxiliary-window outcome.
func (a *AuxWindowService) LastResult() AuxWindowResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last
}

// OpenSetupWizard opens (or focuses) the setup wizard window.
// Prefers Vite-built React native-ui when present under frontend/dist/shell-ui.
func (a *AuxWindowService) OpenSetupWizard() error {
	q := url.Values{}
	if state, platform := setupWizardNativeState(); state != "" {
		// Native-ui decodeDesktopSetupWizardInput requires EXACT keys:
		// locale, state, platform, frame — no extras (not even wails=1).
		q.Set("locale", "en")
		q.Set("state", state)
		q.Set("platform", platform)
		q.Set("frame", "false")
		if doc := nativeUIDocument("setup-wizard"); doc != "" {
			return a.open("setup-wizard", "Set up DSH Desktop",
				"/shell-ui/native-ui/setup-wizard.html?"+q.Encode(), 720, 640, 560, 480)
		}
	}
	q = url.Values{}
	q.Set("wails", "1")
	q.Set("locale", "en")
	return a.open("setup-wizard", "Set up DSH Desktop", "/shell-ui/setup-wizard.html?"+q.Encode(), 720, 640, 560, 480)
}

// OpenProfileSelector opens the profile selection window.
func (a *AuxWindowService) OpenProfileSelector() error {
	profiles := a.ListKnownProfiles()
	q := url.Values{}
	if state := profileSelectorNativeState(profiles); state != "" {
		q.Set("state", state)
	}
	q.Set("locale", "en")
	return a.open("profile-selector", "Switch Profile", a.resolveAuxURL("profile-selector", q), 640, 540, 520, 420)
}

// OpenProfileCreate opens the create-profile window.
func (a *AuxWindowService) OpenProfileCreate() error {
	q := url.Values{}
	q.Set("locale", "en")
	return a.open("profile-create", "Create Profile", a.resolveAuxURL("profile-create", q), 560, 420, 480, 360)
}

// OpenRecovery opens the startup recovery assistant.
// detail is a short failure message shown in the UI.
func (a *AuxWindowService) OpenRecovery(detail string) error {
	a.mu.Lock()
	a.last.Detail = detail
	a.mu.Unlock()
	profiles := a.ListKnownProfiles()
	q := url.Values{}
	if state := recoveryNativeState(detail, profiles); state != "" {
		q.Set("state", state)
	}
	q.Set("locale", "en")
	return a.open("recovery", "DSH Desktop Recovery", a.resolveAuxURL("recovery", q), 800, 720, 680, 560)
}

// resolveAuxURL prefers Vite-built React native-ui; falls back to simplified /shell-ui HTML.
// Assets live under frontend/dist/shell-ui (not "aux") because go:embed rejects the Windows
// reserved device name AUX and would otherwise 404 every auxiliary window.
func (a *AuxWindowService) resolveAuxURL(name string, query url.Values) string {
	if query == nil {
		query = url.Values{}
	}
	if doc := nativeUIDocument(name); doc != "" {
		// Asset-server path only (go:embed frontend/dist). Never file://.
		// Keep caller-supplied query as-is (recovery/profile may include state=).
		if _, ok := query["wails"]; !ok {
			query.Set("wails", "1")
		}
		return "/shell-ui/native-ui/" + name + ".html?" + query.Encode()
	}
	query.Set("wails", "1")
	fallback := "/shell-ui/" + name + ".html"
	if len(query) > 0 {
		return fallback + "?" + query.Encode()
	}
	return fallback
}

// RecoveryDetail returns the failure text for the recovery window.
func (a *AuxWindowService) RecoveryDetail() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.last.Detail != "" {
		return a.last.Detail
	}
	return "Cordis Host failed to start or announce a UI URL."
}

// CompleteSetup records a setup-wizard outcome and closes the wizard.
// action is one of: continue, quit.
func (a *AuxWindowService) CompleteSetup(action string) error {
	switch action {
	case "continue", "quit":
	default:
		return fmt.Errorf("unknown setup action %q", action)
	}
	a.settle("setup-wizard", action, "", "")
	if action == "quit" {
		if a.shell != nil {
			a.shell.Quit()
		}
		return nil
	}
	// Continue → try Host sidecar (or show control UI).
	if a.shell != nil {
		go func() {
			if _, err := a.shell.StartHostSidecar(""); err != nil {
				_ = a.OpenRecovery(err.Error())
			}
		}()
	}
	return nil
}

// CompleteProfileSelection records a profile-selector outcome.
// action: cancel | create | restart | switch; profile is required for switch.
func (a *AuxWindowService) CompleteProfileSelection(action, profile string) error {
	switch action {
	case "cancel", "create", "restart", "switch":
	default:
		return fmt.Errorf("unknown profile action %q", action)
	}
	if action == "switch" && profile == "" {
		return fmt.Errorf("profile name required for switch")
	}
	if action == "create" {
		a.close("profile-selector")
		return a.OpenProfileCreate()
	}
	a.settle("profile-selector", action, profile, "")
	if action == "restart" || action == "switch" {
		if a.shell != nil {
			go func() {
				_ = a.shell.StopHostSidecar()
				if _, err := a.shell.StartHostSidecar(""); err != nil {
					_ = a.OpenRecovery(err.Error())
				}
			}()
		}
	}
	return nil
}

// CompleteProfileCreate records a create-profile outcome.
// action: cancel | create; profile is the new name when creating.
func (a *AuxWindowService) CompleteProfileCreate(action, profile string) error {
	switch action {
	case "cancel", "create":
	default:
		return fmt.Errorf("unknown create-profile action %q", action)
	}
	if action == "create" && profile == "" {
		return fmt.Errorf("profile name required")
	}
	a.settle("profile-create", action, profile, "")
	if action == "create" && a.shell != nil {
		// Hybrid: ask Host relaunch with preferred profile via env file hint.
		a.shell.setPreferredProfile(profile)
		go func() {
			_ = a.shell.StopHostSidecar()
			if _, err := a.shell.StartHostSidecar(""); err != nil {
				_ = a.OpenRecovery(err.Error())
			}
		}()
	}
	return nil
}

// CompleteRecovery records a recovery outcome.
// Supported hybrid actions: restart | safe-mode | quit | profiles | control.
// Checkpoint / plugin-uninstall / diagnostics / config actions return a clear
// Host-controller debt dialog (see docs/wails-migration.md Recovery section).
func (a *AuxWindowService) CompleteRecovery(action string) error {
	action = normalizeRecoveryAction(action)
	switch action {
	case "restart", "safe-mode", "quit", "profiles", "control":
		a.settle("recovery", action, "", "")
		switch action {
		case "quit":
			if a.shell != nil {
				a.shell.Quit()
			}
		case "profiles":
			return a.OpenProfileSelector()
		case "control":
			if a.shell != nil {
				return a.shell.ShowControlUI()
			}
		case "restart", "safe-mode":
			if a.shell != nil {
				if action == "safe-mode" {
					a.shell.setSafeMode(true)
				}
				go func() {
					_ = a.shell.StopHostSidecar()
					if _, err := a.shell.StartHostSidecar(""); err != nil {
						_ = a.OpenRecovery(err.Error())
					}
				}()
			}
		}
		return nil
	case "debt-checkpoint", "debt-uninstall", "debt-diagnostics", "debt-config", "debt-other":
		return a.ReportRecoveryDebt(action)
	default:
		return fmt.Errorf("unknown recovery action %q", action)
	}
}

func normalizeRecoveryAction(action string) string {
	switch action {
	case "enter-safe-mode", "safemode":
		return "safe-mode"
	case "preview-checkpoint", "open-checkpoint", "confirm-checkpoint", "rollback-checkpoint":
		return "debt-checkpoint"
	case "preview-uninstall", "confirm-uninstall", "uninstall-plugin":
		return "debt-uninstall"
	case "export-diagnostics", "show-diagnostics", "save-diagnostics":
		return "debt-diagnostics"
	case "open-settings-document", "open-profile-patch", "open-profile-manifest", "open-profile-directory":
		return "debt-config"
	case "open-terminal", "open-profile-creator", "switch-profile":
		// Profiles creator/switch partially covered elsewhere; mark as debt when
		// invoked from Recovery scheme without profile token authority.
		return "debt-other"
	default:
		return action
	}
}

// ReportRecoveryDebt surfaces precise Host API gaps for Recovery checkpoint/uninstall.
func (a *AuxWindowService) ReportRecoveryDebt(kind string) error {
	title := "Recovery — Host controller required"
	body := recoveryDebtMessage(kind)
	a.mu.Lock()
	a.last = AuxWindowResult{Kind: "recovery", Action: kind, Detail: body}
	a.mu.Unlock()
	return a.OpenInfoDialog(title, body)
}

func recoveryDebtMessage(kind string) string {
	switch kind {
	case "debt-checkpoint":
		return "Checkpoint list / preview / restore is not wired in the Wails hybrid shell.\n\n" +
			"In-process DesktopStartupRecoveryController exists (src/startup-recovery-controller.ts)\n" +
			"but Host↔Wails transport is missing. On Wails recovery, host-main announces\n" +
			"DSH_HOST_RECOVERY_REQUIRED, disposes the controller, and exits.\n\n" +
			"Need Host keep-alive + RPC for:\n" +
			"- snapshot() → checkpoints[] (not Cordis HTTP today)\n" +
			"- previewCheckpointRestore(slotId) / executeCheckpointRestore(previewId)\n" +
			"- generation assert / quiesce around restore\n\n" +
			"Wails hybrid today: restart / safe-mode / quit / profiles / control only.\n" +
			"See docs/wails-migration.md § Recovery controller debt."
	case "debt-uninstall":
		return "Plugin uninstall preview / confirm is not wired in the Wails hybrid shell.\n\n" +
			"Controller methods exist in-process only (no Host↔Wails endpoint):\n" +
			"- previewUninstall(bundleId) + immutable-target rules\n" +
			"- executeUninstall(previewId) under one generationId\n\n" +
			"Do not claim Recovery plugin-tab parity in release notes."
	case "debt-diagnostics":
		return "Diagnostic archive export still needs Electron DesktopDialogWindow / Host controller paths.\n" +
			"Crash-evidence folder is available via Help → Reveal Crash Evidence Folder."
	case "debt-config":
		return "Opening settings.yaml / Profile patch / manifest from Recovery requires Host generation authority.\n" +
			"Use Control UI or a normal Host session after restart when possible."
	default:
		return "This Recovery action needs DesktopStartupRecoveryController state that the Go shell does not own yet.\n" +
			"Supported hybrid actions: restart, safe-mode, quit, profiles, control."
	}
}

// ListKnownProfiles returns profile names discovered from the default DSH home,
// or a placeholder list when none are found (Host still owns authoritative CRUD).
func (a *AuxWindowService) ListKnownProfiles() []string {
	names := discoverLocalProfiles()
	if len(names) == 0 {
		return []string{"default"}
	}
	return names
}

func (a *AuxWindowService) open(name, title, url string, w, h, minW, minH int) error {
	a.mu.Lock()
	app := a.app
	existing := a.windows[name]
	a.mu.Unlock()
	if app == nil {
		return fmt.Errorf("application is not attached")
	}
	if existing != nil {
		existing.Show()
		existing.Focus()
		existing.SetURL(url)
		return nil
	}
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             name,
		Title:            title,
		Width:            w,
		Height:           h,
		MinWidth:         minW,
		MinHeight:        minH,
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              url,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		a.mu.Lock()
		delete(a.windows, name)
		a.mu.Unlock()
	})
	a.mu.Lock()
	a.windows[name] = win
	a.mu.Unlock()
	win.Show()
	win.Focus()
	// Inject scheme bridge after the document begins loading (native-ui + simplified).
	go func() {
		time.Sleep(250 * time.Millisecond)
		win.ExecJS(schemeBridgeJS)
	}()
	return nil
}

func (a *AuxWindowService) close(name string) {
	a.mu.Lock()
	win := a.windows[name]
	delete(a.windows, name)
	a.mu.Unlock()
	if win != nil {
		win.Close()
	}
}

func (a *AuxWindowService) settle(kind, action, profile, detail string) {
	a.mu.Lock()
	a.last = AuxWindowResult{Kind: kind, Action: action, Profile: profile, Detail: detail}
	a.mu.Unlock()
	a.close(kind)
}

// OpenInfoDialog opens a small webview Info dialog with title/message.
// Used on Linux hybrid beds where GTK MessageDialog can be silent from menus.
func (a *AuxWindowService) OpenInfoDialog(title, message string) error {
	if title == "" {
		title = "Info"
	}
	q := url.Values{}
	q.Set("title", title)
	q.Set("message", message)
	q.Set("wails", "1")
	return a.open("info-dialog", title, "/shell-ui/status.html?"+q.Encode(), 560, 420, 420, 280)
}

// CloseInfoDialog closes the Info status window.
func (a *AuxWindowService) CloseInfoDialog() {
	a.close("info-dialog")
}

// ResolveAuxURLForTest exposes resolveAuxURL for unit tests.
func (a *AuxWindowService) ResolveAuxURLForTest(name string, query url.Values) string {
	return a.resolveAuxURL(name, query)
}
