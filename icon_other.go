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
// schema, e.g. Cinnamon/Unity) exposes this directly; anything else falls
// back to the light icon rather than failing.
func isDarkTheme() bool {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "dark")
}
