package main

import (
	_ "embed"

	"golang.org/x/sys/windows/registry"
)

//go:embed assets/icon.ico
var trayIconLight []byte

//go:embed assets/icon-white.ico
var trayIconDark []byte

// trayIcon picks the icon matching the taskbar's current light/dark theme.
func trayIcon() []byte {
	if isDarkTaskbar() {
		return trayIconDark
	}
	return trayIconLight
}

// isDarkTaskbar reads the same registry value Explorer itself uses for
// taskbar theming; if it can't be read (e.g. a very old Windows), the
// light icon is assumed.
func isDarkTaskbar() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	v, _, err := key.GetIntegerValue("SystemUsesLightTheme")
	if err != nil {
		return false
	}
	return v == 0
}
