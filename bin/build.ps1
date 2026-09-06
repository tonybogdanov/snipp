#!/usr/bin/env pwsh
# Builds the Snipp Windows exe locally, mirroring .github/workflows/build.yml.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    go build -ldflags "-H=windowsgui" -o snipp.exe .
    Write-Host "Built $root\snipp.exe"
} finally {
    Pop-Location
}
