package main

import "os"

// doScreenshot picks the capture path based on the session type: X11 can be
// grabbed directly from the X server, but Wayland compositors don't expose
// the framebuffer to clients, so that goes through the xdg-desktop-portal
// Screenshot API instead (works across GNOME, KDE, and wlroots compositors).
func doScreenshot() (string, error) {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return doScreenshotWayland()
	}
	return doScreenshotX11()
}
