//go:build !windows

package main

// initDPIAwareness is a no-op outside Windows: X11 hands clients raw
// framebuffer pixels for the whole (multi-monitor) screen with no OS-level
// DPI virtualization to opt out of, and the Wayland portal returns an
// already-composited, correctly-scaled screenshot.
func initDPIAwareness() {}
