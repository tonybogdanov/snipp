package main

import (
	"fmt"
	"os"
)

// debugEnabled turns on the tracing below when SNIPP_DEBUG is set to
// anything non-empty. Off by default: Snipp is a tray app with nowhere to
// show log output, and the paths worth tracing are ones that otherwise fail
// silently by design (an overlay that can't be shown must never block the
// screenshot itself).
var debugEnabled = os.Getenv("SNIPP_DEBUG") != ""

// debugf writes a trace line to stderr. Run Snipp from a terminal with
// SNIPP_DEBUG=1 to see them:
//
//	SNIPP_DEBUG=1 ~/.local/bin/snipp
//
// On Windows the app is built with -H=windowsgui and has no console
// attached, so these go nowhere unless it's started from a terminal that
// gives it one.
func debugf(format string, args ...any) {
	if !debugEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "snipp: "+format+"\n", args...)
}
