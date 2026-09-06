package main

var procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")

// dpiAwarenessContextPerMonitorAwareV2 is (DPI_AWARENESS_CONTEXT)-4, encoded
// as its 64-bit two's complement (DPI_AWARENESS_CONTEXT is a pseudo-HANDLE).
const dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)

// initDPIAwareness opts the process into per-monitor DPI awareness, so
// GetSystemMetrics/BitBlt see each monitor's true physical resolution
// instead of a virtualized, scaled view of the desktop — otherwise
// screenshots come out downscaled on any scaled display. Silently does
// nothing on Windows versions predating this API (pre-1703).
func initDPIAwareness() {
	if err := procSetProcessDpiAwarenessContext.Find(); err != nil {
		return
	}
	procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2)
}
