package main

import (
	"image"
	"os"
)

// captureScreen picks the capture path based on the session type: X11 can be
// grabbed directly from the X server, but Wayland compositors don't expose
// the framebuffer to clients, so that goes through the xdg-desktop-portal
// Screenshot API instead (works across GNOME, KDE, and wlroots compositors).
func captureScreen() (image.Image, error) {
	if isWayland() {
		return captureScreenWayland()
	}
	return captureScreenX11()
}

// showFreezeOverlay picks the overlay implementation the same way
// captureScreen picks the capture path: X11 gets its own override-redirect
// windows with an explicit keyboard/pointer grab, while Wayland has no
// equivalent client-side mechanism and instead uses the compositor-enforced
// ext-session-lock-v1 protocol.
func showFreezeOverlay(img image.Image) {
	if isWayland() {
		showFreezeOverlayWayland(img)
		return
	}
	showFreezeOverlayX11(img)
}

func isWayland() bool {
	return os.Getenv("XDG_SESSION_TYPE") == "wayland"
}
