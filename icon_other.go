//go:build !windows

package main

import (
	_ "embed"
	"os/exec"
	"strings"
)

//go:embed assets/icon.png
var trayIconLight []byte

//go:embed assets/icon-white.png
var trayIconDark []byte

// trayIcon picks the icon matching the desktop's current light/dark theme.
func trayIcon() []byte {
	if isDarkTheme() {
		return trayIconDark
	}
	return trayIconLight
}

// isDarkTheme is best-effort: GNOME (and anything sharing its settings
// schema, e.g. Cinnamon/Unity) exposes this directly via color-scheme, but
// that key only reflects the newer light/dark switch — many setups (and
// most non-GNOME environments still using GNOME's settings schema) instead
// select a dark variant purely by gtk-theme name (e.g. "Adwaita-dark",
// "Yaru-dark"), leaving color-scheme at "default" while the tray itself is
// dark. Check both, falling back to the light icon rather than failing.
func isDarkTheme() bool {
	if gsettingsContains("color-scheme", "dark") {
		return true
	}
	return gsettingsContains("gtk-theme", "dark")
}

func gsettingsContains(key, substr string) bool {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", key).Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), substr)
}
