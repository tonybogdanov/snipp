package main

import (
	"bytes"
	"time"

	"fyne.io/systray"
)

func main() {
	initDPIAwareness()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(trayIcon())
	systray.SetTitle("Snipp")
	systray.SetTooltip("Snipp")

	screenshot := systray.AddMenuItem("Take Screenshot", "Take a screenshot")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Quit Snipp")

	systray.SetOnTapped(takeScreenshot)

	go watchTrayTheme()

	go func() {
		for {
			select {
			case <-screenshot.ClickedCh:
				takeScreenshot()
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// watchTrayTheme keeps the tray icon matching the OS light/dark theme while
// Snipp keeps running — trayIcon() is otherwise only ever read once, at
// startup, so a theme switch (or autostart racing ahead of the desktop
// finishing loading its theme) would leave the wrong-contrast icon stuck
// until the app was restarted. Neither Windows nor Linux's theme lookup is
// cheap enough to call every tick, so this just polls at a human timescale.
func watchTrayTheme() {
	current := trayIcon()
	for range time.Tick(5 * time.Second) {
		next := trayIcon()
		if !bytes.Equal(current, next) {
			current = next
			systray.SetIcon(current)
		}
	}
}

func onExit() {
	// nothing to clean up yet
}

func takeScreenshot() {
	path, err := doScreenshot()
	if err != nil {
		alert("Snipp", "Screenshot failed: "+err.Error())
		return
	}
	alert("Snipp", "Saved to "+path)
}
