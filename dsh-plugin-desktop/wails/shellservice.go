package main

import (
	"fmt"
	"os"
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
}

func NewShellService() *ShellService {
	return &ShellService{}
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
	s.mu.Unlock()
	if app != nil {
		app.Quit()
	} else {
		os.Exit(0)
	}
}
