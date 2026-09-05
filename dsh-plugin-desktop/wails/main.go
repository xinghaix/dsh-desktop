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

	app := application.New(application.Options{
		Name:        "DSH Desktop",
		Description: "DSH Desktop native shell (Wails v3)",
		Services: []application.Service{
			application.NewService(shell),
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
	if initialURL != "/" {
		shell.setInitialHostURL(initialURL)
	} else if !*noHost {
		go func() {
			ready, err := shell.StartHostSidecar("")
			if err != nil {
				log.Printf("dsh-wails-shell: host sidecar: %v", err)
				app.Dialog.Error().
					SetTitle("Cordis Host failed").
					SetMessage(err.Error()).
					Show()
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

	setupApplicationMenu(app, shell, window)
	setupSystemTray(app, shell, window)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func setupApplicationMenu(app *application.App, shell *ShellService, window *application.WebviewWindow) {
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

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("About DSH Desktop (Wails)").OnClick(func(ctx *application.Context) {
		app.Dialog.Info().
			SetTitle("DSH Desktop").
			SetMessage("Native shell powered by Go + Wails v3.\nElectron BrowserWindow/Tray/Dialog surface is being replaced here;\nCordis Host remains a Node sidecar during the hybrid migration.").
			Show()
	})

	app.Menu.Set(menu)
	_ = window
}

func setupSystemTray(app *application.App, shell *ShellService, window *application.WebviewWindow) {
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
