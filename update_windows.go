package main

// installerAsset is the release asset that installs Snipp on this platform,
// and tmpPattern the os.CreateTemp pattern used when downloading it — the
// .exe suffix is load-bearing on Windows, which won't execute a file
// without it.
const (
	installerAsset = "snipp-installer.exe"
	tmpPattern     = "snipp-installer-*.exe"
)
