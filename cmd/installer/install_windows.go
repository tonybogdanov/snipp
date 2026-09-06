package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// install places the embedded binary in %LOCALAPPDATA%\Snipp, registers it
// to autostart at login, kills any already-running instance so the new
// binary takes effect immediately, and starts it. Renaming the old exe out
// of the way before overwriting works because Windows opens running
// executables with share-delete, not share-write: the rename succeeds even
// while the old process is still executing from it.
func install() {
	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Snipp")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return
	}
	target := filepath.Join(installDir, "snipp.exe")
	old := target + ".old"

	os.Remove(old)
	os.Rename(target, old)

	if err := os.WriteFile(target, appBinary, 0o755); err != nil {
		os.Rename(old, target)
		return
	}

	exec.Command("taskkill", "/F", "/IM", "snipp.exe").Run()
	os.Remove(old)

	registerAutostart(target)

	exec.Command(target).Start()
}

func registerAutostart(target string) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	key.SetStringValue("Snipp", `"`+target+`"`)
}
