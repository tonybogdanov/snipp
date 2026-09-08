package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// install puts the embedded binary in place, registers Snipp to autostart
// at login, stops any running instance so the binary in place is the one
// running, and starts it.
//
// It installs what it carries and nothing else: keeping it current is the
// app's own job, via the tray's "Check for Updates". That keeps the
// installer free of any network dependency — it works offline, and behaves
// the same every time it's run.
//
// It returns the message to show the user, so a failure explains itself
// rather than leaving the dialog claiming success.
func install() string {
	dir, err := installDir()
	if err != nil {
		return "Snipp could not be installed:\n" + err.Error()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "Snipp could not be installed:\n" + err.Error()
	}
	target := filepath.Join(dir, binaryName)

	if err := placeBinary(target, appBinary); err != nil {
		return "Snipp could not be installed:\n" + err.Error()
	}

	killRunning()
	registerAutostart(target)
	exec.Command(target).Start()

	return "Snipp is installed and running."
}
