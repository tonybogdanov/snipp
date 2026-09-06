#!/usr/bin/env pwsh
# Builds the Snipp Windows exe and its installer locally, mirroring
# .github/workflows/build.yml.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path artifacts | Out-Null
    go build -ldflags "-H=windowsgui" -o artifacts/snipp.exe .
    Write-Host "Built $root\artifacts\snipp.exe"

    $payload = "cmd/installer/payload/windows.bin"
    Copy-Item artifacts/snipp.exe $payload -Force
    try {
        go build -ldflags "-H=windowsgui" -o artifacts/snipp-installer.exe ./cmd/installer
        Write-Host "Built $root\artifacts\snipp-installer.exe"
    } finally {
        git checkout -- $payload
    }
} finally {
    Pop-Location
}
