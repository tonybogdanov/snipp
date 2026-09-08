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
// equivalent client-side mechanism and uses whichever of the
// compositor-enforced ext-session-lock-v1 or a fullscreen xdg-shell
// toplevel the compositor offers — see showFreezeOverlayWayland.
func showFreezeOverlay(img image.Image) {
	debugf("overlay: version=%s XDG_SESSION_TYPE=%q XDG_CURRENT_DESKTOP=%q WAYLAND_DISPLAY=%q DISPLAY=%q capture=%v",
		version,
		os.Getenv("XDG_SESSION_TYPE"), os.Getenv("XDG_CURRENT_DESKTOP"),
		os.Getenv("WAYLAND_DISPLAY"), os.Getenv("DISPLAY"), img.Bounds())

	if isWayland() {
		showFreezeOverlayWayland(img)
		return
	}
	showFreezeOverlayX11(img)
}

// isWayland reports whether this is a Wayland session. XDG_SESSION_TYPE is
// set by the login manager and is the canonical answer, but it's missing in
// sessions started outside one (a compositor launched from a TTY, a nested
// compositor); WAYLAND_DISPLAY is set by the compositor itself, so it
// catches those.
func isWayland() bool {
	return os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("WAYLAND_DISPLAY") != ""
}
