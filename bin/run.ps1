#!/usr/bin/env pwsh
# Builds and runs the Snipp app on Windows.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    go run -ldflags "-H=windowsgui" .
} finally {
    Pop-Location
}
