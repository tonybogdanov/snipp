package main

import (
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
