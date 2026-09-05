package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const profilePatchFilename = "cordis.patch.yml"

// resolveDesktopUserDataDir mirrors Electron userData for DSH Desktop.
func resolveDesktopUserDataDir() string {
	if home := strings.TrimSpace(os.Getenv("DSH_DESKTOP_USER_DATA")); home != "" {
		return home
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(userHome, "Library", "Application Support", "DSH Desktop")
	case "windows":
		if appdata := strings.TrimSpace(os.Getenv("APPDATA")); appdata != "" {
			return filepath.Join(appdata, "DSH Desktop")
		}
		return filepath.Join(userHome, "AppData", "Roaming", "DSH Desktop")
	default:
		config := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if config == "" {
			config = filepath.Join(userHome, ".config")
		}
		return filepath.Join(config, "DSH Desktop")
	}
}

// resolveLegacyDshHome is the Cordis ~/.dsh (or DSH_HOME) tree used by older shells.
func resolveLegacyDshHome() string {
	home := strings.TrimSpace(os.Getenv("DSH_HOME"))
	if home != "" {
		return home
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(userHome, ".dsh")
}

// discoverLocalProfiles lists profile directory names from Desktop userData, then legacy ~/.dsh.
// Full Cordis profile materialization remains Host-owned; this is a shell hint.
func discoverLocalProfiles() []string {
	roots := []string{}
	if ud := resolveDesktopUserDataDir(); ud != "" {
		roots = append(roots, filepath.Join(ud, "profiles"))
	}
	if legacy := resolveLegacyDshHome(); legacy != "" {
		roots = append(roots, filepath.Join(legacy, "profiles"))
	}
	seen := map[string]struct{}{}
	var names []string
	for _, profilesDir := range roots {
		entries, err := os.ReadDir(profilesDir)
		if err != nil {
			continue
		}
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
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func resolveActiveProfileName(preferred string) string {
	preferred = strings.TrimSpace(preferred)
	names := discoverLocalProfiles()
	if preferred != "" {
		for _, n := range names {
			if n == preferred {
				return preferred
			}
		}
	}
	if len(names) > 0 {
		return names[0]
	}
	return "default"
}

func resolveProfileDir(profileName string) string {
	profileName = resolveActiveProfileName(profileName)
	if ud := resolveDesktopUserDataDir(); ud != "" {
		p := filepath.Join(ud, "profiles", profileName)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if legacy := resolveLegacyDshHome(); legacy != "" {
		p := filepath.Join(legacy, "profiles", profileName)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		return p
	}
	if ud := resolveDesktopUserDataDir(); ud != "" {
		return filepath.Join(ud, "profiles", profileName)
	}
	return ""
}

// recoveryConfigPath returns a best-effort local path for Recovery config actions.
// kind: settingsDocument | profilePatch | profileManifest | profileDirectory
func recoveryConfigPath(kind, preferredProfile string) string {
	ud := resolveDesktopUserDataDir()
	profileDir := resolveProfileDir(preferredProfile)
	switch kind {
	case "settingsDocument", "open-settings-document":
		if ud != "" {
			return filepath.Join(ud, "settings.yaml")
		}
	case "profilePatch", "open-profile-patch":
		if profileDir != "" {
			return filepath.Join(profileDir, profilePatchFilename)
		}
	case "profileManifest", "open-profile-manifest":
		if profileDir != "" {
			return filepath.Join(profileDir, "package.json")
		}
	case "profileDirectory", "open-profile-directory":
		return profileDir
	}
	return ""
}
