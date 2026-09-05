package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	hostURL := flag.String("host-url", envOr("DSH_HOST_URL", ""), "Cordis Host UI URL to load (e.g. http://127.0.0.1:PORT/)")
	startHidden := flag.Bool("hidden", false, "Start with the main window hidden (tray still runs)")
	noHost := flag.Bool("no-host", false, "Do not auto-start Cordis Host; show embedded control UI only")
	flag.Parse()

	sidecar := NewHostSidecar()
	shell := NewShellService(sidecar)
	aux := NewAuxWindowService(shell)
	bridge := NewBridgeService(shell)
	notifier := notifications.New()
	caps := NewCapabilitiesService(shell, notifier)
	crash := NewCrashEvidenceService()
	shell.attachAux(aux)
	shell.attachBridge(bridge)
	sidecar.AttachBridge(bridge)
	sidecar.AttachAux(aux)
	sidecar.AttachCaps(caps)
	caps.attachCrash(crash)
	aux.attachCaps(caps)

	app := application.New(application.Options{
		Name:        "DSH Desktop",
		Description: "DSH Desktop native shell (Wails v3)",
		Services: []application.Service{
			application.NewService(shell),
			application.NewService(aux),
			application.NewService(bridge),
			application.NewService(notifier),
			application.NewService(caps),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	initialURL := "/"
	if u := strings.TrimSpace(*hostURL); u != "" {
		initialURL = u
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "DSH Desktop — Wails shell",
		Width:            1280,
		Height:           800,
		MinWidth:         800,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              initialURL,
		Hidden:           *startHidden,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})
	shell.attach(app, window)
	aux.attach(app)
	bridge.attach(app)
	caps.attach(app)
	log.Printf("dsh-wails-shell: %s", crash.BeginRun(currentPackageVersion()))
	defer crash.MarkClean()
	defer func() {
		if r := recover(); r != nil {
			if p := crash.WritePanicDump(r); p != "" {
				log.Printf("dsh-wails-shell: panic dump written to %s", p)
			}
			panic(r)
		}
	}()
	_ = caps.ApplyPlatformIdentity()
	if initialURL != "/" {
		shell.setInitialHostURL(initialURL)
	} else if !*noHost {
		go func() {
			ready, err := shell.StartHostSidecar("")
			if err != nil {
				log.Printf("dsh-wails-shell: host sidecar: %v", err)
				if openErr := aux.OpenRecovery(err.Error()); openErr != nil {
					app.Dialog.Error().
						SetTitle("Cordis Host failed").
						SetMessage(err.Error()).
						Show()
				}
				return
			}
			log.Printf("dsh-wails-shell: loaded Cordis Host UI %s", ready)
		}()
	}

	// Close-to-tray: keep the process alive for tray / Host sidecar lifecycle.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	setupApplicationMenu(app, shell, aux, bridge, caps, window)
	setupSystemTray(app, shell, caps, window)

	if os.Getenv("DSH_WAILS_AUTO_RECOVERY") != "" {
		go func() {
			time.Sleep(800 * time.Millisecond)
			if err := aux.OpenRecovery(os.Getenv("DSH_WAILS_AUTO_RECOVERY")); err != nil {
				log.Printf("dsh-wails-shell: auto recovery: %v", err)
			}
		}()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// showMenuStatus prefers AuxWindowService.OpenInfoDialog (status.html) so menu
// feedback is visible on Linux hybrid beds where GTK MessageDialog can be silent.
// Falls back to native Dialog.Info if the aux window cannot open.

// isDialogCancelled reports native file/directory picker cancel (empty path on
// Linux/macOS; "cancelled by user" error from the Windows common-file-dialog path).
func isDialogCancelled(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cancel")
}

func showMenuStatus(aux *AuxWindowService, app *application.App, title, message string) {
	if aux != nil {
		if err := aux.OpenInfoDialog(title, message); err == nil {
			return
		}
	}
	if app != nil {
		app.Dialog.Info().SetTitle(title).SetMessage(message).Show()
	}
}

func setupApplicationMenu(app *application.App, shell *ShellService, aux *AuxWindowService, bridge *BridgeService, caps *CapabilitiesService, window *application.WebviewWindow) {
	menu := app.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}

	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Open Directory…").OnClick(func(ctx *application.Context) {
		path, err := shell.OpenDirectoryDialog()
		if err != nil {
			if isDialogCancelled(err) {
				showMenuStatus(aux, app, "Open Directory", "Cancelled — no directory selected.")
				return
			}
			showMenuStatus(aux, app, "Open Directory", "Failed:\n"+err.Error())
			return
		}
		if path == "" {
			showMenuStatus(aux, app, "Open Directory", "Cancelled — no directory selected.")
			return
		}
		showMenuStatus(aux, app, "Selected Directory", path)
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Setup Wizard…").OnClick(func(ctx *application.Context) {
		if err := aux.OpenSetupWizard(); err != nil {
			showMenuStatus(aux, app, "Setup Wizard", "Failed:\n"+err.Error())
		}
	})
	fileMenu.Add("Switch Profile…").OnClick(func(ctx *application.Context) {
		if err := aux.OpenProfileSelector(); err != nil {
			showMenuStatus(aux, app, "Profiles", "Failed:\n"+err.Error())
		}
	})
	fileMenu.Add("Recovery Assistant…").OnClick(func(ctx *application.Context) {
		if err := aux.OpenRecovery(""); err != nil {
			showMenuStatus(aux, app, "Recovery", "Failed:\n"+err.Error())
		}
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Quit DSH Desktop").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Show Window").OnClick(func(ctx *application.Context) {
		shell.ShowWindow()
	})
	viewMenu.Add("Hide Window").OnClick(func(ctx *application.Context) {
		shell.HideWindow()
	})
	viewMenu.AddSeparator()
	viewMenu.Add("Embedded Control UI").OnClick(func(ctx *application.Context) {
		_ = shell.ShowControlUI()
	})
	viewMenu.Add("Reload Host URL").OnClick(func(ctx *application.Context) {
		url := shell.CurrentURL()
		if url == "" {
			showMenuStatus(aux, app, "No Host URL", "Set -host-url / DSH_HOST_URL or call ShellService.LoadHostURL first.")
			return
		}
		_ = shell.LoadHostURL(url)
	})

	toolsMenu := menu.AddSubmenu("Tools")
	toolsMenu.Add("Open Terminal…").OnClick(func(ctx *application.Context) {
		if err := caps.OpenTerminal(""); err != nil {
			showMenuStatus(aux, app, "Terminal", "Failed:\n"+err.Error())
		}
	})
	toolsMenu.Add("Export Text…").OnClick(func(ctx *application.Context) {
		path, err := caps.ExportTextFile("dsh-shell-note.txt", "DSH Wails shell export\n")
		if err != nil {
			showMenuStatus(aux, app, "Export", "Failed:\n"+err.Error())
			return
		}
		if path != "" {
			showMenuStatus(aux, app, "Exported", path)
		}
	})
	toolsMenu.Add("Send Test Notification").OnClick(func(ctx *application.Context) {
		if err := caps.NotifyAttention("DSH Desktop", "Native notification from the Wails shell."); err != nil {
			showMenuStatus(aux, app, "Notification", "Failed:\n"+err.Error())
		}
	})
	toolsMenu.Add("Request Dock / Taskbar Attention").OnClick(func(ctx *application.Context) {
		_ = caps.RequestUserAttention(1)
		showMenuStatus(aux, app, "Attention", caps.DockAttentionStatus())
	})
	toolsMenu.Add("Clear Dock Attention").OnClick(func(ctx *application.Context) {
		_ = caps.ClearUserAttention()
		showMenuStatus(aux, app, "Dock Attention Cleared", caps.DockAttentionStatus())
	})
	toolsMenu.Add("Check for Updates…").OnClick(func(ctx *application.Context) {
		res := caps.CheckForUpdates()
		showMenuStatus(aux, app, "Updates", res.Status+"\n"+res.Detail+"\ncurrent="+res.CurrentHint+"\nlatest="+res.LatestHint)
	})
	toolsMenu.Add("Download / Install Update…").OnClick(func(ctx *application.Context) {
		res := caps.DownloadAndInstallUpdate()
		showMenuStatus(aux, app, "Update download", res.Status+"\n"+res.Detail)
	})
	toolsMenu.AddSeparator()
	toolsMenu.Add("LAN HTTPS Status").OnClick(func(ctx *application.Context) {
		showMenuStatus(aux, app, "LAN HTTPS", caps.LanHttpsStatus())
	})

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("Capabilities Status").OnClick(func(ctx *application.Context) {
		showMenuStatus(aux, app, "Capabilities", caps.CapabilitiesStatus())
	})
	helpMenu.Add("Auth / Renderer Header…").OnClick(func(ctx *application.Context) {
		showMenuStatus(aux, app, "Auth / Renderer Header", bridge.BridgeStatus()+"\n\n"+bridge.PlatformAuthNotes())
	})
	helpMenu.Add("Crash Evidence Status").OnClick(func(ctx *application.Context) {
		showMenuStatus(aux, app, "Crash evidence", caps.CrashEvidenceStatus())
	})
	helpMenu.Add("Reveal Crash Evidence Folder").OnClick(func(ctx *application.Context) {
		if err := caps.RevealCrashEvidenceFolder(); err != nil {
			showMenuStatus(aux, app, "Crash evidence", "Reveal failed:\n"+err.Error()+"\n\n"+caps.CrashEvidenceStatus())
			return
		}
		showMenuStatus(aux, app, "Crash evidence", "Opened folder.\n\n"+caps.CrashEvidenceStatus())
	})
	helpMenu.Add("Export Diagnostic Archive…").OnClick(func(ctx *application.Context) {
		res, err := caps.ExportDiagnosticArchive(true)
		if err != nil {
			showMenuStatus(aux, app, "Diagnostics", "Export failed:\n"+err.Error()+"\n\n"+caps.CrashEvidenceStatus())
			return
		}
		msg := "Diagnostic archive ready.\n\n" + res.Detail
		if res.Path != "" {
			msg += "\n\nPath:\n" + res.Path
		}
		showMenuStatus(aux, app, "Diagnostics archive", msg)
	})
	helpMenu.Add("Platform Identity").OnClick(func(ctx *application.Context) {
		id := caps.PlatformIdentity()
		showMenuStatus(aux, app, "Platform identity",
			"appId="+id.AppID+"\n"+id.AppUserModelID+"\n"+id.Dock+"\n"+id.CrashReporter+"\n"+id.PackagedUpdates+"\napplied="+id.Applied)
	})
	helpMenu.Add("About DSH Desktop (Wails)").OnClick(func(ctx *application.Context) {
		showMenuStatus(aux, app, "DSH Desktop",
			"Native shell powered by Go + Wails v3.\nPrimary product path: Wails + Node Cordis Host.\nElectron main.ts is LAST-RESORT when forced by ABI (see docs/wails-migration.md).")
	})

	app.Menu.Set(menu)
	_ = window
}

func setupSystemTray(app *application.App, shell *ShellService, caps *CapabilitiesService, window *application.WebviewWindow) {
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(icons.SystrayLight)
	}
	tray.SetTooltip("DSH Desktop")
	caps.attachTray(tray)

	trayMenu := app.NewMenu()
	trayMenu.Add("Show DSH Desktop").OnClick(func(ctx *application.Context) {
		shell.ShowWindow()
	})
	trayMenu.Add("Hide Window").OnClick(func(ctx *application.Context) {
		shell.HideWindow()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Control UI").OnClick(func(ctx *application.Context) {
		_ = shell.ShowControlUI()
	})
	trayMenu.Add("Open Terminal…").OnClick(func(ctx *application.Context) {
		_ = caps.OpenTerminal("")
	})
	trayMenu.Add("Check for Updates…").OnClick(func(ctx *application.Context) {
		_ = caps.CheckForUpdates()
		shell.ShowInfoDialog("Updates", caps.LastUpdateCheck().Detail)
	})
	trayMenu.Add("About").OnClick(func(ctx *application.Context) {
		shell.ShowInfoDialog(
			"DSH Desktop",
			"Wails v3 native shell\nHybrid: Cordis Host still boots via Node until rewritten.",
		)
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)

	tray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			shell.ShowWindow()
		}
	})
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
