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
// Prefers Vite-built React native-ui when lib/native-ui is present.
func (a *AuxWindowService) OpenSetupWizard() error {
	return a.open("setup-wizard", "Set up DSH Desktop", a.resolveAuxURL("setup-wizard", nil), 720, 640, 560, 480)
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

// resolveAuxURL prefers Vite-built React native-ui; falls back to simplified /aux HTML.
func (a *AuxWindowService) resolveAuxURL(name string, query url.Values) string {
	if query == nil {
		query = url.Values{}
	}
	query.Set("wails", "1")
	if doc := nativeUIDocument(name); doc != "" {
		// Asset-server path only (go:embed frontend/dist). Never file://.
		return "/aux/native-ui/" + name + ".html?" + query.Encode()
	}
	fallback := "/aux/" + name + ".html"
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
// action: restart | safe-mode | quit | profiles | control.
func (a *AuxWindowService) CompleteRecovery(action string) error {
	switch action {
	case "restart", "safe-mode", "quit", "profiles", "control":
	default:
		return fmt.Errorf("unknown recovery action %q", action)
	}
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
	return a.open("info-dialog", title, "/aux/status.html?"+q.Encode(), 520, 320, 420, 240)
}

// CloseInfoDialog closes the Info status window.
func (a *AuxWindowService) CloseInfoDialog() {
	a.close("info-dialog")
}

// ResolveAuxURLForTest exposes resolveAuxURL for unit tests.
func (a *AuxWindowService) ResolveAuxURLForTest(name string, query url.Values) string {
	return a.resolveAuxURL(name, query)
}
