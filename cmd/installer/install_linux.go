//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// run shows a pulsating zenity progress dialog while the install happens,
// then updates that same dialog's text in place to report the outcome —
// rather than closing it and opening a second confirmation dialog, which
// looked like two different, inconsistent windows. Once installed, the user
// dismisses it via the window's own close button. The exception is an
// outcome too long for that one-line text (an install failure and its
// cause), which gets its own message dialog. If zenity isn't available the
// outcome goes out as a notification rather than being swallowed.
//
// Known limitation: the dock/taskbar icon during install is zenity's own
// generic icon, not Snipp's. --window-icon only sets the icon painted
// inside the dialog and its _NET_WM_ICON pixmap; docks like GNOME Shell's
// instead resolve the taskbar icon from a .desktop file matched by the
// window's app id, which zenity hardcodes to itself — not something a
// flag can override. Fixing it for real means dropping zenity for a GUI
// toolkit we control (e.g. cgo + GTK), which isn't worth it for a
// few-second, install-time-only dialog.
func run() {
	zenity, err := exec.LookPath("zenity")
	if err != nil {
		// No dialog toolkit: install anyway, and report the outcome
		// through a desktop notification if one is available.
		notify(install())
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
		notify(install())
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

	message := install()

	if !strings.Contains(message, "\n") {
		stdin.Write([]byte("# " + message + "\n"))
		stdin.Close()
		return
	}

	// The progress dialog's text is a single line fed through its stdin, so
	// a multi-line message (an install failure and its cause) gets its own
	// dialog instead.
	stdin.Close()
	if progress.Process != nil {
		progress.Process.Kill()
		progress.Wait()
	}
	info(zenity, message)
}

// info shows the outcome in a plain message dialog, used for messages too
// long for the progress dialog's one-line text.
func info(zenity, message string) {
	args := []string{"--info", "--title=Snipp", "--text=" + message}
	if iconPath := writeTempIcon(); iconPath != "" {
		defer os.Remove(iconPath)
		args = append(args, "--window-icon="+iconPath)
	}
	exec.Command(zenity, args...).Run()
}

// notify is the fallback for desktops without zenity: no dialog to update,
// so the outcome goes out as a notification instead of vanishing.
func notify(message string) {
	exec.Command("notify-send", "Snipp", message).Run()
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

// binaryName is what the app is called once installed.
const binaryName = "snipp"

// installDir is the per-user install location; no root needed.
func installDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// placeBinary writes the embedded binary over the installed one. Writing
// to a temp file and renaming over the target is safe even if the old
// binary is still running from it (Linux keeps the old inode open).
func placeBinary(target string, binary []byte) error {
	tmp := target + ".new"

	if err := os.WriteFile(tmp, binary, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// killRunning stops any running instance, so the binary in place is the
// one running.
func killRunning() {
	exec.Command("pkill", "-x", "snipp").Run()
}

func registerAutostart(target string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

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
