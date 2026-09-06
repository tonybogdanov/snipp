package main

import _ "embed"

//go:embed assets/icon.ico
var trayIconBytes []byte

func trayIcon() []byte {
	return trayIconBytes
}
