package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// AuxWindowService owns setup / profile / recovery webview windows that
// Electron previously opened as BrowserWindow HTML UIs under src/native-ui/.
// These Wails windows are hybrid stand-ins: they talk to Go via bindings and
// cover shell startup flows while Cordis Host still owns full profile CRUD.
type pendingRecoveryConfirm struct {
	Kind      string // checkpoint | uninstall
	PreviewID string
	Subject   string
	Detail    string
	Danger    bool
}

type AuxWindowService struct {
	mu               sync.Mutex
	app              *application.App
	shell            *ShellService
	caps             *CapabilitiesService
	windows          map[string]*application.WebviewWindow
	last             AuxWindowResult
	recoveryRPC      *RecoveryRpcClient
	pendingConfirm   *pendingRecoveryConfirm
	preferredProfile string
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

func (a *AuxWindowService) attachCaps(caps *CapabilitiesService) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.caps = caps
}

func (a *AuxWindowService) capabilities() *CapabilitiesService {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.caps
}

// AttachRecoveryRPC stores the Host Recovery RPC client for checkpoint/uninstall.
func (a *AuxWindowService) AttachRecoveryRPC(client *RecoveryRpcClient) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recoveryRPC = client
}

func (a *AuxWindowService) recoveryClient() *RecoveryRpcClient {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.recoveryRPC != nil {
		return a.recoveryRPC
	}
	if a.shell != nil && a.shell.sidecar != nil {
		return a.shell.sidecar.RecoveryRPC()
	}
	return nil
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
	var snapshot *RecoverySnapshot
	if client := a.recoveryClient(); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if snap, err := client.Snapshot(ctx); err == nil {
			snapshot = snap
		}
	}
	q := url.Values{}
	if state := recoveryNativeState(detail, profiles, snapshot); state != "" {
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
// action may include checkpoint/uninstall when Host Recovery RPC is attached.
// targetId carries scheme ?id= (slotId, bundleId, or previewId).
func (a *AuxWindowService) CompleteRecovery(action string, targetId string) error {
	id := targetId
	action = normalizeRecoveryAction(action)
	switch action {
	case "restart", "safe-mode", "quit", "profiles", "control":
		if a.shell != nil && (action == "restart" || action == "safe-mode" || action == "quit") {
			// Suppress auto-relaunch before Host may exit from /v1/complete quiesce.
			a.shell.ExpectHostExit()
		}
		if client := a.recoveryClient(); client != nil && (action == "restart" || action == "safe-mode" || action == "quit") {
			// Ordered shutdown: Host quiesce (fiber dispose) via /v1/complete, then StopHostSidecar.
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			_ = client.Complete(ctx, action)
			cancel()
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
					// Coarse process stop remains; /v1/complete already attempted generation quiesce.
					_ = a.shell.StopHostSidecar()
					if _, err := a.shell.StartHostSidecar(""); err != nil {
						_ = a.OpenRecovery(err.Error())
					}
				}()
			}
		}
		return nil
	case "preview-checkpoint", "rollback-checkpoint":
		return a.handleCheckpointPreview(id)
	case "confirm-checkpoint":
		return a.handleCheckpointExecute(id)
	case "open-checkpoint":
		return a.handleOpenCheckpoint(id)
	case "preview-uninstall", "uninstall-plugin":
		return a.handleUninstallPreview(id)
	case "confirm-uninstall":
		return a.handleUninstallExecute(id)
	case "export-diagnostics", "show-diagnostics", "save-diagnostics":
		return a.handleRecoveryDiagnostics(action)
	case "open-settings-document", "open-profile-patch", "open-profile-manifest", "open-profile-directory":
		return a.handleRecoveryConfig(action)
	case "open-terminal":
		return a.handleRecoveryTerminal()
	case "open-profile-creator":
		return a.OpenProfileCreate()
	case "switch-profile":
		return a.handleRecoverySwitchProfile(id)
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
	default:
		return action
	}
}

func (a *AuxWindowService) handleCheckpointPreview(slotID string) error {
	client := a.recoveryClient()
	if client == nil {
		return a.ReportRecoveryDebt("debt-checkpoint")
	}
	if slotID == "" {
		return a.OpenInfoDialog("Checkpoint preview", "Missing checkpoint slot id.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	preview, err := client.PreviewCheckpointRestore(ctx, slotID)
	if err != nil {
		return a.OpenInfoDialog("Checkpoint preview failed", err.Error())
	}
	title, message, confirmLabel := checkpointConfirmCopy(preview)
	a.storePendingConfirm(&pendingRecoveryConfirm{
		Kind:      "checkpoint",
		PreviewID: preview.PreviewID,
		Subject:   preview.SlotID,
		Detail:    message,
	})
	return a.OpenConfirmDialog(title, message, confirmLabel, "Cancel", false)
}

func checkpointConfirmCopy(preview *checkpointPreview) (title, message, confirmLabel string) {
	title = "Confirm rollback"
	confirmLabel = "Restore checkpoint"
	captured := preview.CapturedAt
	if captured == "" {
		captured = "unknown time"
	}
	slot := preview.SlotID
	if slot == "" {
		slot = "(unknown slot)"
	}
	message = fmt.Sprintf("Restore checkpoint %s?\n\nCaptured: %s\nPreview expires: %s\n\nThis replaces the current profile state. Restart Desktop after restore to finish applying changes.",
		slot, captured, preview.ExpiresAt)
	return title, message, confirmLabel
}

func (a *AuxWindowService) handleCheckpointExecute(previewID string) error {
	client := a.recoveryClient()
	if client == nil {
		return a.ReportRecoveryDebt("debt-checkpoint")
	}
	if previewID == "" {
		return a.OpenInfoDialog("Checkpoint confirm", "Missing restore preview id.")
	}
	quiesceNote := a.quiesceHostBestEffort(8 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := client.ExecuteCheckpointRestore(ctx, previewID)
	if err != nil {
		return a.OpenInfoDialog("Checkpoint restore failed", err.Error())
	}
	a.settle("recovery", "confirm-checkpoint", "", fmt.Sprintf("%v", result))
	return a.OpenInfoDialog("Checkpoint restored",
		fmt.Sprintf("%v\n\n%s\n\nRestart Desktop to finish applying the restore.", result, quiesceNote))
}

func (a *AuxWindowService) handleOpenCheckpoint(slotID string) error {
	client := a.recoveryClient()
	if client == nil {
		return a.ReportRecoveryDebt("debt-checkpoint")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := client.OpenCheckpoint(ctx, slotID); err != nil {
		return a.OpenInfoDialog("Open checkpoint", err.Error())
	}
	return a.OpenInfoDialog("Open checkpoint", "Host acknowledged openCheckpoint for "+slotID)
}

func (a *AuxWindowService) handleUninstallPreview(bundleID string) error {
	client := a.recoveryClient()
	if client == nil {
		return a.ReportRecoveryDebt("debt-uninstall")
	}
	if bundleID == "" {
		return a.OpenInfoDialog("Plugin uninstall", "Missing bundle id.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	preview, err := client.PreviewUninstall(ctx, bundleID)
	if err != nil {
		return a.OpenInfoDialog("Plugin uninstall preview failed", err.Error())
	}
	title, message, confirmLabel := uninstallConfirmCopy(preview)
	a.storePendingConfirm(&pendingRecoveryConfirm{
		Kind:      "uninstall",
		PreviewID: preview.PreviewID,
		Subject:   preview.PackageName,
		Detail:    message,
		Danger:    true,
	})
	return a.OpenConfirmDialog(title, message, confirmLabel, "Cancel", true)
}

func uninstallConfirmCopy(preview *uninstallPreview) (title, message, confirmLabel string) {
	title = "Confirm uninstall"
	confirmLabel = "Uninstall"
	pkg := preview.PackageName
	if pkg == "" {
		pkg = "(unknown package)"
	}
	message = fmt.Sprintf("Uninstall %s?\n\nPreview expires: %s\n\nThis removes the plugin from the active profile. Restart Desktop if the Host UI still lists it.",
		pkg, preview.ExpiresAt)
	return title, message, confirmLabel
}

func (a *AuxWindowService) handleUninstallExecute(previewID string) error {
	client := a.recoveryClient()
	if client == nil {
		return a.ReportRecoveryDebt("debt-uninstall")
	}
	if previewID == "" {
		return a.OpenInfoDialog("Plugin uninstall", "Missing uninstall preview id.")
	}
	quiesceNote := a.quiesceHostBestEffort(8 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := client.ExecuteUninstall(ctx, previewID)
	if err != nil {
		return a.OpenInfoDialog("Plugin uninstall failed", err.Error())
	}
	a.settle("recovery", "confirm-uninstall", "", fmt.Sprintf("%v", result))
	return a.OpenInfoDialog("Plugin uninstalled",
		fmt.Sprintf("%v\n\n%s\n\nRestart Desktop if the Host UI still lists the plugin.", result, quiesceNote))
}

func (a *AuxWindowService) storePendingConfirm(pending *pendingRecoveryConfirm) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pendingConfirm = pending
}

func (a *AuxWindowService) takePendingConfirm() *pendingRecoveryConfirm {
	a.mu.Lock()
	defer a.mu.Unlock()
	pending := a.pendingConfirm
	a.pendingConfirm = nil
	return pending
}

// OpenConfirmDialog opens a webview Confirm/Cancel dialog (Linux-safe; GTK Question can be silent).
func (a *AuxWindowService) OpenConfirmDialog(title, message, confirmLabel, cancelLabel string, danger bool) error {
	if title == "" {
		title = "Confirm"
	}
	if confirmLabel == "" {
		confirmLabel = "Confirm"
	}
	if cancelLabel == "" {
		cancelLabel = "Cancel"
	}
	q := url.Values{}
	q.Set("title", title)
	q.Set("message", message)
	q.Set("confirm", confirmLabel)
	q.Set("cancel", cancelLabel)
	q.Set("wails", "1")
	if danger {
		q.Set("danger", "1")
	}
	return a.open("confirm-dialog", title, "/shell-ui/confirm.html?"+q.Encode(), 560, 420, 420, 280)
}

// CloseConfirmDialog closes the confirm dialog window.
func (a *AuxWindowService) CloseConfirmDialog() {
	a.close("confirm-dialog")
}

// CompleteConfirmDialog handles Confirm/Cancel from confirm.html.
// response: confirm | cancel (also accepts yes/ok/accept).
func (a *AuxWindowService) CompleteConfirmDialog(response string) error {
	a.CloseConfirmDialog()
	accepted := false
	switch response {
	case "confirm", "yes", "ok", "accept":
		accepted = true
	case "cancel", "no", "dismiss", "":
		accepted = false
	default:
		return fmt.Errorf("unknown confirm response %q", response)
	}
	pending := a.takePendingConfirm()
	if !accepted || pending == nil {
		return nil
	}
	switch pending.Kind {
	case "checkpoint":
		return a.handleCheckpointExecute(pending.PreviewID)
	case "uninstall":
		return a.handleUninstallExecute(pending.PreviewID)
	default:
		return fmt.Errorf("unknown pending confirm kind %q", pending.Kind)
	}
}


// quiesceHostBestEffort asks Host Recovery RPC to dispose the Cordis Host fiber.
// Real API: DesktopStartupGeneration.quiesceForRecovery via POST /v1/quiesce.
// There is no drain-generations / wait-for-idle / cancel-in-flight Host surface.
func (a *AuxWindowService) quiesceHostBestEffort(timeout time.Duration) string {
	client := a.recoveryClient()
	if client == nil {
		return "quiesce=skipped (no Recovery RPC)"
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	res, err := client.Quiesce(ctx)
	if err != nil {
		return "quiesce=rpc-error: " + err.Error()
	}
	if res == nil {
		return "quiesce=empty-response"
	}
	if res.Detail != "" {
		return res.Detail
	}
	if res.OK {
		return "quiesce=ok"
	}
	return "quiesce=not-ok"
}

func (a *AuxWindowService) handleRecoveryDiagnostics(action string) error {
	caps := a.capabilities()
	if caps == nil {
		return a.ReportRecoveryDebt("debt-diagnostics")
	}
	offerSave := action == "export-diagnostics" || action == "save-diagnostics"
	res, err := caps.ExportDiagnosticArchive(offerSave)
	if err != nil {
		revealErr := caps.RevealCrashEvidenceFolder()
		msg := "Diagnostic archive export failed:\n" + err.Error()
		if revealErr == nil {
			msg += "\n\nOpened crash-evidence folder as a fallback.\n\n" + caps.CrashEvidenceStatus()
		} else {
			msg += "\n\nCrash-evidence reveal also failed: " + revealErr.Error() + "\n\n" + caps.CrashEvidenceStatus()
		}
		return a.OpenInfoDialog("Diagnostics", msg)
	}
	title := "Diagnostics archive"
	detail := "Created Host/Electron-format diagnostic archive.\n\n" + res.Detail
	if res.SavedPath != "" {
		detail += "\n\nSaved copy: " + res.SavedPath
	} else if res.Cancelled {
		detail += "\n\nSave dialog cancelled; archive kept at:\n" + res.Path
	} else {
		detail += "\n\nArchive path:\n" + res.Path
	}
	a.mu.Lock()
	a.last = AuxWindowResult{Kind: "recovery", Action: action, Detail: res.Path}
	a.mu.Unlock()
	return a.OpenInfoDialog(title, detail)
}

func (a *AuxWindowService) handleRecoveryConfig(action string) error {
	caps := a.capabilities()
	a.mu.Lock()
	preferred := a.preferredProfile
	a.mu.Unlock()
	path := recoveryConfigPath(action, preferred)
	if path == "" {
		return a.ReportRecoveryDebt("debt-config")
	}
	if _, err := os.Stat(path); err != nil {
		return a.OpenInfoDialog("Configuration",
			fmt.Sprintf("Path not found yet:\n%s\n\nHost generation may create it on a normal boot.", path))
	}
	if caps == nil {
		return a.ReportRecoveryDebt("debt-config")
	}
	selectFile := action != "open-profile-directory"
	if err := caps.RevealInFileManager(path, selectFile); err != nil {
		return a.OpenInfoDialog("Configuration", "Reveal failed:\n"+err.Error()+"\n\n"+path)
	}
	a.mu.Lock()
	a.last = AuxWindowResult{Kind: "recovery", Action: action, Detail: path}
	a.mu.Unlock()
	return nil
}

func (a *AuxWindowService) handleRecoveryTerminal() error {
	caps := a.capabilities()
	if caps == nil {
		return a.ReportRecoveryDebt("debt-other")
	}
	a.mu.Lock()
	preferred := a.preferredProfile
	a.mu.Unlock()
	workdir := recoveryConfigPath("open-profile-directory", preferred)
	if workdir == "" {
		workdir = resolveDesktopUserDataDir()
	}
	if err := caps.OpenTerminal(workdir); err != nil {
		return a.OpenInfoDialog("Terminal", err.Error())
	}
	return nil
}

func (a *AuxWindowService) handleRecoverySwitchProfile(profileName string) error {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return a.OpenProfileSelector()
	}
	a.mu.Lock()
	a.preferredProfile = profileName
	a.mu.Unlock()
	if a.shell != nil {
		a.shell.setPreferredProfile(profileName)
	}
	a.settle("recovery", "switch-profile", profileName, "Preferred profile recorded for next Host start.")
	return a.OpenInfoDialog("Profile selected",
		fmt.Sprintf("Preferred profile set to %q.\nUse Restart in Recovery (or Start Host sidecar) to boot with it.", profileName))
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
		return "Checkpoint Host Recovery RPC is not attached for this session.\n\n" +
			"Transport exists (DSH_HOST_RECOVERY_RPC + /v1/snapshot|checkpoint/*).\n" +
			"Host must keep DesktopStartupRecoveryController alive and announce the RPC URL.\n" +
			"APIs: snapshot(), previewCheckpointRestore, executeCheckpointRestore.\n" +
			"See docs/wails-migration.md § Recovery."
	case "debt-uninstall":
		return "Plugin uninstall Host Recovery RPC is not attached for this session.\n\n" +
			"APIs: previewUninstall(bundleId), executeUninstall(previewId).\n" +
			"Immutable-target / generation assert still enforced inside the Host controller."
	case "debt-diagnostics":
		return "CapabilitiesService is not attached, so diagnostic archive export cannot run.\n" +
			"When caps are attached: Recovery RPC POST /v1/diagnostics/export (exportDesktopDiagnostics) " +
			"or CLI node lib/host-main.js --export-diagnostics, then optional SaveFileDialog + reveal.\n" +
			"Help → Export Diagnostic Archive… uses the same path."
	case "debt-config":
		return "Could not resolve local settings.yaml / cordis.patch.yml / profile manifest under Desktop userData.\n" +
			"Host generation still owns authoritative paths; try after a normal boot creates them."
	default:
		return "This Recovery action is not wired for the hybrid shell yet.\n" +
			"Supported: restart, safe-mode, quit, profiles, control, checkpoint/uninstall (RPC+confirm),\n" +
			"diagnostic archive zip (RPC/CLI), local config reveal, terminal, profile creator/switch."
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
