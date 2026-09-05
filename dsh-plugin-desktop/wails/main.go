package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"runtime"
	"strings"

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
	shell.attachAux(aux)
	shell.attachBridge(bridge)
	sidecar.AttachBridge(bridge)
	sidecar.AttachAux(aux)
	sidecar.AttachCaps(caps)

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

	if err := app.Run(); err != nil {
		log.Fatal(err)
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
			app.Dialog.Error().SetTitle("Open Directory").SetMessage(err.Error()).Show()
			return
		}
		if path == "" {
			return
		}
		app.Dialog.Info().SetTitle("Selected Directory").SetMessage(path).Show()
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Setup Wizard…").OnClick(func(ctx *application.Context) {
		if err := aux.OpenSetupWizard(); err != nil {
			app.Dialog.Error().SetTitle("Setup Wizard").SetMessage(err.Error()).Show()
		}
	})
	fileMenu.Add("Switch Profile…").OnClick(func(ctx *application.Context) {
		if err := aux.OpenProfileSelector(); err != nil {
			app.Dialog.Error().SetTitle("Profiles").SetMessage(err.Error()).Show()
		}
	})
	fileMenu.Add("Recovery Assistant…").OnClick(func(ctx *application.Context) {
		if err := aux.OpenRecovery(""); err != nil {
			app.Dialog.Error().SetTitle("Recovery").SetMessage(err.Error()).Show()
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
			app.Dialog.Warning().
				SetTitle("No Host URL").
				SetMessage("Set -host-url / DSH_HOST_URL or call ShellService.LoadHostURL first.").
				Show()
			return
		}
		_ = shell.LoadHostURL(url)
	})

	toolsMenu := menu.AddSubmenu("Tools")
	toolsMenu.Add("Open Terminal…").OnClick(func(ctx *application.Context) {
		if err := caps.OpenTerminal(""); err != nil {
			app.Dialog.Error().SetTitle("Terminal").SetMessage(err.Error()).Show()
		}
	})
	toolsMenu.Add("Export Text…").OnClick(func(ctx *application.Context) {
		path, err := caps.ExportTextFile("dsh-shell-note.txt", "DSH Wails shell export\n")
		if err != nil {
			app.Dialog.Error().SetTitle("Export").SetMessage(err.Error()).Show()
			return
		}
		if path != "" {
			app.Dialog.Info().SetTitle("Exported").SetMessage(path).Show()
		}
	})
	toolsMenu.Add("Send Test Notification").OnClick(func(ctx *application.Context) {
		if err := caps.NotifyAttention("DSH Desktop", "Native notification from the Wails shell."); err != nil {
			app.Dialog.Error().SetTitle("Notification").SetMessage(err.Error()).Show()
		}
	})
	toolsMenu.Add("Check for Updates…").OnClick(func(ctx *application.Context) {
		res := caps.CheckForUpdates()
		app.Dialog.Info().SetTitle("Updates").SetMessage(res.Status + "\n" + res.Detail + "\ncurrent=" + res.CurrentHint + "\nlatest=" + res.LatestHint).Show()
	})
	toolsMenu.Add("Download / Install Update…").OnClick(func(ctx *application.Context) {
		res := caps.DownloadAndInstallUpdate()
		app.Dialog.Info().SetTitle("Update download").SetMessage(res.Status + "\n" + res.Detail).Show()
	})
	toolsMenu.AddSeparator()
	toolsMenu.Add("LAN HTTPS Status").OnClick(func(ctx *application.Context) {
		app.Dialog.Info().SetTitle("LAN HTTPS").SetMessage(caps.LanHttpsStatus()).Show()
	})

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("Auth / Renderer Header…").OnClick(func(ctx *application.Context) {
		app.Dialog.Info().SetTitle("Auth bridge").SetMessage(bridge.BridgeStatus() + "\n\n" + bridge.PlatformAuthNotes()).Show()
	})
	helpMenu.Add("About DSH Desktop (Wails)").OnClick(func(ctx *application.Context) {
		app.Dialog.Info().
			SetTitle("DSH Desktop").
			SetMessage("Native shell powered by Go + Wails v3.\nElectron BrowserWindow/Tray/Dialog surface is being replaced here;\nCordis Host remains a Node sidecar during the hybrid migration.").
			Show()
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
