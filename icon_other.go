//go:build !windows

package main

import _ "embed"

//go:embed assets/icon.png
var trayIconBytes []byte

func trayIcon() []byte {
	return trayIconBytes
}
