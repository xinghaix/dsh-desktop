package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// discoverLocalProfiles lists profile directory names under the default DSH home.
// Full Cordis profile materialization remains Host-owned; this is a shell hint.
func discoverLocalProfiles() []string {
	home := strings.TrimSpace(os.Getenv("DSH_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		home = filepath.Join(userHome, ".dsh")
	}
	profilesDir := filepath.Join(home, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		if strings.ContainsAny(name, `/\`+"\x00\r\n") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
