//go:build !windows

package main

import "os/exec"

// alert shows a standard OS message box with the given text, via zenity
// (GTK/Qt desktops) falling back to notify-send.
func alert(title, text string) {
	if err := exec.Command("zenity", "--info", "--title", title, "--text", text).Run(); err == nil {
		return
	}
	exec.Command("notify-send", title, text).Run()
}
