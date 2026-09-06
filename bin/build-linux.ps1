#!/usr/bin/env pwsh
# Cross-compiles the Snipp Linux binary locally, mirroring the linux job in
# .github/workflows/build.yml. No Linux toolchain needed (pure Go, no cgo).

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    go build -o snipp-linux .
    Write-Host "Built $root\snipp-linux"
} finally {
    Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}
