//go:build !windows

package main

// installerAsset is the release asset that installs Snipp on this platform,
// and tmpPattern the os.CreateTemp pattern used when downloading it.
const (
	installerAsset = "snipp-installer"
	tmpPattern     = "snipp-installer-*"
)
