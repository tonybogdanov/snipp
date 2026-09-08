package main

// version is the short commit hash this binary was built from, stamped in
// at build time by bin/build-windows.ps1 / bin/build-linux.sh (and so by
// the CI jobs that run them) via -ldflags "-X main.version=...". It matches
// the tag of the GitHub release built from that same commit, which is what
// the updater compares against. Plain `go build .` leaves it as "dev",
// which never matches a release — an unstamped build always sees itself as
// out of date.
var version = "dev"
