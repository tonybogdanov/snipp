package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// versionFile records, next to the installed binary, the release tag the
// binary came from. Nothing else knows it — the binary is downloaded, not
// embedded, so there's no build-time value to read back out of this
// installer — and without it a re-run couldn't tell an up-to-date install
// from one that needs replacing.
const versionFile = "version.txt"

// install brings this machine onto the latest release: it checks what that
// release is, downloads its binary unless the installed one already came
// from it, registers Snipp to autostart at login, stops any running
// instance so the binary in place is the one running, and starts it.
//
// It returns the message to show the user — for the good paths as well as
// the bad ones, since with the binary downloaded rather than embedded, "no
// network" is an ordinary outcome that deserves a plain explanation rather
// than a silent no-op.
func install() string {
	dir, err := installDir()
	if err != nil {
		return "Snipp could not be installed:\n" + err.Error()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "Snipp could not be installed:\n" + err.Error()
	}
	target := filepath.Join(dir, binaryName)

	latest, err := latestVersion()
	if err != nil {
		if errors.Is(err, errOffline) {
			return offlineMessage(target)
		}
		return "Snipp could not be installed:\n" + err.Error()
	}

	if current(dir, target) == latest {
		// Already the current build; skip the download but still restart it
		// from the installed binary, so running this installer always
		// leaves Snipp running and registered.
		restart(target)
		return "Snipp is already up to date (" + latest + ") and running."
	}

	binary, err := fetchBinary()
	if err != nil {
		if errors.Is(err, errOffline) {
			return offlineMessage(target)
		}
		return "Snipp could not be installed:\n" + err.Error()
	}

	if err := placeBinary(target, binary); err != nil {
		return "Snipp could not be installed:\n" + err.Error()
	}
	os.WriteFile(filepath.Join(dir, versionFile), []byte(latest), 0o644)

	restart(target)
	return "Snipp " + latest + " is installed and running."
}

// current reports the release the installed binary came from, or "" if
// Snipp isn't installed here or its version wasn't recorded (an install
// from before this file existed) — either way, something to download.
func current(dir, target string) string {
	if _, err := os.Stat(target); err != nil {
		return ""
	}
	recorded, err := os.ReadFile(filepath.Join(dir, versionFile))
	if err != nil {
		return ""
	}
	return string(recorded)
}

// restart registers autostart and swaps the running instance for the one
// at target.
func restart(target string) {
	killRunning()
	registerAutostart(target)
	exec.Command(target).Start()
}

// offlineMessage explains the failure in terms of what it means for the
// user, which depends on whether they already have a working Snipp.
func offlineMessage(target string) string {
	if _, err := os.Stat(target); err == nil {
		restart(target)
		return "Couldn't reach GitHub to check for a newer Snipp.\n" +
			"Your installed version was left in place and is running.\n\n" +
			"Check your internet connection and try again."
	}
	return "Couldn't reach GitHub to download Snipp.\n\n" +
		"Check your internet connection and try again."
}
