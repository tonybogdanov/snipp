#!/usr/bin/env pwsh
# Builds the Snipp Windows exe locally, mirroring .github/workflows/build.yml.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path artifacts | Out-Null
    go build -ldflags "-H=windowsgui" -o artifacts/snipp.exe .
    Write-Host "Built $root\artifacts\snipp.exe"
} finally {
    Pop-Location
}
