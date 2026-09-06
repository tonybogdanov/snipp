//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// run shows a pulsating zenity progress dialog while the install happens,
// then updates that same dialog's text in place to report completion —
// rather than closing it and opening a second confirmation dialog, which
// looked like two different, inconsistent windows. Once installed, the user
// dismisses it via the window's own close button. If zenity isn't available
// it just installs silently rather than failing.
func run() {
	zenity, err := exec.LookPath("zenity")
	if err != nil {
		install()
		return
	}

	iconPath := writeTempIcon()

	args := []string{"--progress", "--pulsate", "--no-cancel",
		"--title=Snipp", "--text=Installing Snipp..."}
	if iconPath != "" {
		args = append(args, "--window-icon="+iconPath)
	}

	progress := exec.Command(zenity, args...)
	stdin, err := progress.StdinPipe()
	if err != nil || progress.Start() != nil {
		install()
		return
	}

	if iconPath != "" {
		// zenity reads --window-icon once at startup; give it a moment
		// before cleaning up the temp file rather than racing it.
		go func() {
			time.Sleep(2 * time.Second)
			os.Remove(iconPath)
		}()
	}

	install()

	stdin.Write([]byte("# Snipp is installed and running.\n"))
	stdin.Close()
}

// writeTempIcon spills the embedded icon to a temp file, since zenity's
// --window-icon only accepts a file path, not raw bytes.
func writeTempIcon() string {
	tmp, err := os.CreateTemp("", "snipp-icon-*.png")
	if err != nil {
		return ""
	}
	if _, err := tmp.Write(appIcon); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return ""
	}
	tmp.Close()
	return tmp.Name()
}

// install places the embedded binary in ~/.local/bin, registers it to
// autostart at login via the XDG autostart spec, kills any already-running
// instance so the new binary takes effect immediately, and starts it.
// Writing to a temp file and renaming over the target is safe even if the
// old binary is still running from it (Linux keeps the old inode open).
func install() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	installDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return
	}
	target := filepath.Join(installDir, "snipp")
	tmp := target + ".new"

	if err := os.WriteFile(tmp, appBinary, 0o755); err != nil {
		return
	}
	if err := os.Rename(tmp, target); err != nil {
		return
	}

	exec.Command("pkill", "-x", "snipp").Run()

	registerAutostart(home, target)

	exec.Command(target).Start()
}

func registerAutostart(home, target string) {
	entry := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Snipp\n" +
		"Exec=" + target + "\n" +
		"X-GNOME-Autostart-enabled=true\n"

	autostartDir := filepath.Join(home, ".config", "autostart")
	if err := os.MkdirAll(autostartDir, 0o755); err == nil {
		os.WriteFile(filepath.Join(autostartDir, "snipp.desktop"), []byte(entry), 0o644)
	}

	appsDir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0o755); err == nil {
		os.WriteFile(filepath.Join(appsDir, "snipp.desktop"), []byte(entry), 0o644)
	}
}
